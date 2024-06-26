package authapi

import (
	"errors"
	"net/http"

	"github.com/capdale/was/api"
	"github.com/capdale/was/auth"
	baselogger "github.com/capdale/was/logger"
	"github.com/capdale/was/types/claimer"
	"github.com/gin-gonic/gin"
)

var logger = baselogger.Logger

type database interface {
	DeleteUserAccount(claimer *claimer.Claimer) error
}

type AuthAPI struct {
	DB   database
	Auth *auth.Auth
}

var (
	ErrStateNotEqual    = errors.New("state is not equal")
	ErrNoValidEmail     = errors.New("no valid email")
	ErrAlreayExistEmail = errors.New("already exist email")
)

var (
	AccessDenied = gin.H{
		"message": "access denied",
	}
)

func New(database database, auth *auth.Auth) *AuthAPI {
	return &AuthAPI{
		DB:   database,
		Auth: auth,
	}
}

func (a *AuthAPI) SetBlacklistHandler(ctx *gin.Context) {
	accessToken := ctx.Param("access_token")
	refreshToken := ctx.Param("refresh_token")
	if err := a.Auth.BlackToken(&accessToken, &refreshToken); err != nil {
		ctx.Status(http.StatusBadRequest)
		logger.ErrorWithCTX(ctx, "black token", err)
		return
	}
	ctx.Status(http.StatusOK)
}

func (a *AuthAPI) DeleteUserAccountHandler(ctx *gin.Context) {
	claimer := api.MustGetClaimer(ctx)
	if err := a.DB.DeleteUserAccount(claimer); err != nil {
		ctx.Status(http.StatusInternalServerError)
		logger.ErrorWithCTX(ctx, "delete user account", err)
		return
	}
	ctx.Status(http.StatusAccepted)
}

type RefreshTokenReq struct {
	RefreshToken *string `json:"refresh_token" binding:"required"`
}

func (a *AuthAPI) RefreshTokenHandler(ctx *gin.Context) {
	form := new(RefreshTokenReq)
	err := ctx.BindJSON(form)
	if err != nil {
		api.BasicBadRequestError(ctx)
		logger.ErrorWithCTX(ctx, "binding form", err)
		return
	}

	userAgent := ctx.Request.UserAgent()

	newToken, newRefreshToken, err := a.Auth.RefreshToken(*form.RefreshToken, &userAgent)
	if err != nil {
		api.BasicUnAuthorizedError(ctx)
		logger.ErrorWithCTX(ctx, "refresh token failed", err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"access_token":  newToken,
		"refresh_token": newRefreshToken,
	})
}

func (a *AuthAPI) WhoAmIHandler(ctx *gin.Context) {
}
