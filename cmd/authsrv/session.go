package main

import (
	"errors"
	"log"
	"time"

	"github.com/DmitriyPrischep/backend-WAO/pkg/auth"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const (
	expiration = 10 * time.Minute
)

type SessionManager struct {
	// Definition DateBase
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		//Initialize DataBase
	}
}

func generateToken(in *auth.UserData) (token string, err error) {
	rawToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": in.Login,
		"agent":    in.Agent,
		"exp":      time.Now().Add(expiration).Unix(),
	})
	log.Printf("SECRET KEY (%T) %s", secret, secret)

	tokenString, err := rawToken.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// Create JWT for user
func (sm *SessionManager) Create(ctx context.Context, in *auth.UserData) (*auth.Token, error) {
	log.Println("call Create", in)
	token, err := generateToken(in)
	if err != nil {
		log.Println("Token does not create:", err)
		return nil, err
	}
	id := &auth.Token{
		Value: token,
	}
	//Add token to White list of DataBase
	log.Println("Token create: ", id.Value)
	return id, nil

}

// Check validation of token
func (sm *SessionManager) Check(ctx context.Context, tokenString *auth.Token) (*auth.UserData, error) {
	log.Println("call Check", tokenString)
	var err error
	token, err := jwt.Parse(tokenString.Value, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, err
		}
		return []byte(secret), nil
	})
	if err != nil {
		log.Printf("Unexpected signing method: %v", token.Header["alg"])
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		log.Println("CLAIMS:", claims)
		username, ok := claims["username"]
		if !ok {
			return nil, errors.New("Bad claims: field 'username' not exist")
		}

		user := &auth.UserData{
			Login: username.(string),
		}
		log.Println("Hooooray, Token is exist")
		return user, nil
	}
	return nil, grpc.Errorf(codes.NotFound, "session not found")
}

// Delete token
func (sm *SessionManager) Delete(ctx context.Context, in *auth.Token) (*auth.Nothing, error) {
	log.Println("call Delete", in)
	//Delete from WhiteList of DataBase
	return &auth.Nothing{Null: true}, nil
}
