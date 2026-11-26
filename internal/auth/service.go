package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Mousynano/rbac-backend/internal/platform/emailpolicy"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrCredentials = errors.New("invalid credentials")

type Mailer interface {
	Send(context.Context, string, string, string) error
}
type Service struct {
	DB       *pgxpool.Pool
	Mailer   Mailer
	HMACKey  []byte
	Resolver emailpolicy.Resolver
}
type Pending struct {
	Token   string
	Purpose string
}

// StartPasswordFlow validates credentials and starts a pending email challenge.
func (s Service) StartPasswordFlow(ctx context.Context, email, password, purpose string) (Pending, error) {
	var err error
	email, err = emailpolicy.Normalize(email)
	if err != nil {
		return Pending{}, ErrCredentials
	}
	if err = s.allowStart(ctx, email, purpose); err != nil {
		return Pending{}, err
	}
	if err = s.checkEmail(ctx, email); err != nil {
		return Pending{}, err
	}
	if purpose == "register" {
		if !validPassword(password) {
			return Pending{}, ErrCredentials
		}
	} else if purpose != "login" {
		return Pending{}, ErrCredentials
	}

	var userID, passwordHash string
	err = s.DB.QueryRow(ctx, `SELECT id::text, COALESCE(password_hash,'') FROM users WHERE email=$1 AND status='active'`, email).Scan(&userID, &passwordHash)
	if purpose == "register" && err == nil {
		return Pending{}, ErrCredentials
	}
	if purpose == "login" {
		if err != nil || passwordHash == "" || !VerifyPassword(password, passwordHash) {
			return Pending{}, ErrCredentials
		}
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Pending{}, fmt.Errorf("find account: %w", err)
	}
	if purpose == "register" {
		passwordHash, err = HashPassword(password)
		if err != nil {
			return Pending{}, err
		}
	}
	return s.createChallenge(ctx, email, userID, passwordHash, purpose)
}

func (s Service) allowStart(ctx context.Context, email, purpose string) error {
	h := hmac.New(sha256.New, s.HMACKey)
	h.Write([]byte(email))
	key := h.Sum(nil)
	var count int
	err := s.DB.QueryRow(ctx, `INSERT INTO auth_rate_limits(scope,key_hash,window_start,count) VALUES($1,$2,date_trunc('minute',now())-make_interval(mins => (extract(minute from now())::int % 15)),1) ON CONFLICT(scope,key_hash,window_start) DO UPDATE SET count=auth_rate_limits.count+1 RETURNING count`, "challenge:"+purpose, key).Scan(&count)
	if err != nil {
		return fmt.Errorf("record challenge rate: %w", err)
	}
	if count > 5 {
		return ErrCredentials
	}
	return nil
}

func (s Service) createChallenge(ctx context.Context, email, userID, passwordHash, purpose string) (Pending, error) {
	code, err := NewOTP()
	if err != nil {
		return Pending{}, err
	}
	var tokenBytes [32]byte
	if _, err = rand.Read(tokenBytes[:]); err != nil {
		return Pending{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes[:])
	tokenHash := sha256.Sum256(tokenBytes[:])
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return Pending{}, fmt.Errorf("begin challenge: %w", err)
	}
	defer tx.Rollback(ctx)
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO auth_challenges(user_id,email,purpose,pending_token_hash,pending_password_hash,otp_digest,sent_at,expires_at,family_expires_at) VALUES(NULLIF($1,'')::uuid,$2,$3,$4,$5,$6,now(),now()+interval '5 minutes',now()+interval '15 minutes') RETURNING id::text`, userID, email, purpose, tokenHash[:], nullable(passwordHash), OTPDigest(s.HMACKey, "", purpose, code)).Scan(&id)
	if err != nil {
		return Pending{}, fmt.Errorf("create challenge: %w", err)
	}
	// Replace digest with challenge-bound value after obtaining its database ID.
	_, err = tx.Exec(ctx, `UPDATE auth_challenges SET otp_digest=$2 WHERE id=$1::uuid`, id, OTPDigest(s.HMACKey, id, purpose, code))
	if err != nil {
		return Pending{}, fmt.Errorf("bind challenge otp: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return Pending{}, fmt.Errorf("commit challenge: %w", err)
	}
	if err = s.Mailer.Send(ctx, email, "Your verification code", fmt.Sprintf("Your code is %s. It expires in 5 minutes.", code)); err != nil {
		_, _ = s.DB.Exec(ctx, `UPDATE auth_challenges SET invalidated_at=now(), otp_digest=NULL, pending_password_hash=NULL WHERE id=$1::uuid`, id)
		return Pending{}, fmt.Errorf("send verification email: %w", err)
	}
	return Pending{Token: token, Purpose: purpose}, nil
}

// Verify consumes an OTP once and atomically creates the account and session.
func (s Service) Verify(ctx context.Context, token, code string) (UserProfile, string, error) {
	if len(code) != 6 {
		return UserProfile{}, "", ErrCredentials
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return UserProfile{}, "", ErrCredentials
		}
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != 32 {
		return UserProfile{}, "", ErrCredentials
	}
	tokenHash := sha256.Sum256(raw)
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return UserProfile{}, "", fmt.Errorf("begin verify: %w", err)
	}
	defer tx.Rollback(ctx)
	var id, email, purpose, pw string
	var digest []byte
	var attempts int
	err = tx.QueryRow(ctx, `SELECT id::text,email,purpose,COALESCE(pending_password_hash,''),otp_digest,attempts FROM auth_challenges WHERE pending_token_hash=$1 AND consumed_at IS NULL AND invalidated_at IS NULL AND expires_at>now() AND family_expires_at>now() FOR UPDATE`, tokenHash[:]).Scan(&id, &email, &purpose, &pw, &digest, &attempts)
	if err != nil {
		return UserProfile{}, "", ErrCredentials
	}
	if attempts >= 5 {
		return UserProfile{}, "", ErrCredentials
	}
	if err = s.checkEmail(ctx, email); err != nil {
		return UserProfile{}, "", err
	}
	var blocked bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM email_blocklist b WHERE (b.kind='email' AND b.value=$1) OR (b.kind='domain' AND (split_part($1,'@',2)=b.value OR split_part($1,'@',2) LIKE '%.'||b.value))) OR EXISTS(SELECT 1 FROM disposable_email_domains d WHERE split_part($1,'@',2)=d.domain OR split_part($1,'@',2) LIKE '%.'||d.domain)`, email).Scan(&blocked); err != nil {
		return UserProfile{}, "", err
	}
	if blocked {
		return UserProfile{}, "", ErrCredentials
	}
	if !VerifyOTPDigest(s.HMACKey, id, purpose, code, digest) {
		if _, e := tx.Exec(ctx, `UPDATE auth_challenges SET attempts=attempts+1 WHERE id=$1::uuid`, id); e != nil {
			return UserProfile{}, "", e
		}
		if e := tx.Commit(ctx); e != nil {
			return UserProfile{}, "", e
		}
		return UserProfile{}, "", ErrCredentials
	}
	var userID string
	if purpose == "register" {
		err = tx.QueryRow(ctx, `INSERT INTO users(email,password_hash,email_verified_at) VALUES($1,$2,now()) RETURNING id::text`, email, pw).Scan(&userID)
	} else {
		err = tx.QueryRow(ctx, `SELECT id::text FROM users WHERE email=$1 AND status='active'`, email).Scan(&userID)
	}
	if err != nil {
		return UserProfile{}, "", fmt.Errorf("resolve verified account: %w", err)
	}
	var sessionBytes [32]byte
	if _, err = rand.Read(sessionBytes[:]); err != nil {
		return UserProfile{}, "", err
	}
	session := base64.RawURLEncoding.EncodeToString(sessionBytes[:])
	sessionHash := sha256.Sum256(sessionBytes[:])
	_, err = tx.Exec(ctx, `INSERT INTO sessions(user_id,token_hash,expires_at) VALUES($1::uuid,$2,now()+interval '24 hours')`, userID, sessionHash[:])
	if err != nil {
		return UserProfile{}, "", fmt.Errorf("create session: %w", err)
	}
	if _, err = tx.Exec(ctx, `UPDATE auth_challenges SET consumed_at=now(),otp_digest=NULL,pending_password_hash=NULL WHERE id=$1::uuid`, id); err != nil {
		return UserProfile{}, "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return UserProfile{}, "", fmt.Errorf("commit verification: %w", err)
	}
	return UserProfile{ID: userID, Email: email, Status: "active"}, session, nil
}

func (s Service) checkEmail(ctx context.Context, email string) error {
	var blocked bool
	err := s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM email_blocklist b WHERE (b.kind='email' AND b.value=$1) OR (b.kind='domain' AND (split_part($1,'@',2)=b.value OR split_part($1,'@',2) LIKE '%.'||b.value))) OR EXISTS(SELECT 1 FROM disposable_email_domains d WHERE split_part($1,'@',2)=d.domain OR split_part($1,'@',2) LIKE '%.'||d.domain)`, email).Scan(&blocked)
	if err != nil {
		return fmt.Errorf("check email policy: %w", err)
	}
	if blocked {
		return ErrCredentials
	}
	if s.Resolver == nil {
		s.Resolver = emailpolicy.SystemResolver{}
	}
	return emailpolicy.CheckDomain(ctx, s.Resolver, strings.SplitN(email, "@", 2)[1])
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nowUTC is kept local to the package for consistent cookie/session expiry handling.
var nowUTC = func() time.Time { return time.Now().UTC() }
