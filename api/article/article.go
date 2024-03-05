package articleAPI

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/capdale/was/auth"
	baselogger "github.com/capdale/was/logger"
	"github.com/capdale/was/model"
	"github.com/capdale/was/types/binaryuuid"
	"github.com/gin-gonic/gin"
)

var logger = baselogger.Logger

type storage interface {
	GetArticleJPG(ctx context.Context, uuid binaryuuid.UUID) (*[]byte, error)
	UploadArticleJPGs(ctx context.Context, uuids *[]binaryuuid.UUID, readers *[]io.Reader) error
}

type database interface {
	IsCollectionOwned(userId int64, collectionUUIDs *[]binaryuuid.UUID) error
	GetArticleLinkIdsByUserId(userId int64, offset int, limit int) (*[]binaryuuid.UUID, error)
	GetUserIdByUUID(userUUID binaryuuid.UUID) (int64, error)
	GetUserIdByName(username string) (int64, error)
	GetArticle(linkId binaryuuid.UUID) (*model.ArticleAPI, error)
	CreateNewArticle(userId int64, title string, content string, collectionUUIDs *[]binaryuuid.UUID, imageUUIDs *[]binaryuuid.UUID, collectionOrder *[]uint8) error
	HasAccessPermissionArticleImage(claimerUUID *binaryuuid.UUID, articleImageUUID binaryuuid.UUID) error
}

type ArticleAPI struct {
	d       database
	Storage storage
}

func New(d database, storage storage) *ArticleAPI {
	return &ArticleAPI{
		d:       d,
		Storage: storage,
	}
}

var ErrInvalidForm = errors.New("form is invalid")

type createArticleForm struct {
	Article      articleForm             `form:"article" binding:"required"`
	ImageHeaders []*multipart.FileHeader `form:"image[]"`
}

type articleForm struct {
	Title           string           `form:"title" json:"title" binding:"required,min=4,max=32"`
	Content         string           `form:"content" json:"content" binding:"required,min=8,max=512"`
	CollectionInfos []collectionInfo `form:"collections" json:"collections" binding:"required,min=1"`
}

type collectionInfo struct {
	UUID  string `form:"uuid" binding:"required,uuid"`
	Order *uint8 `form:"order" binding:"required"`
}

var ErrInvalidOrder = errors.New("invalid order")

func (a *ArticleAPI) CreateArticleHandler(ctx *gin.Context) {
	claims := ctx.MustGet("claims").(*auth.AuthClaims)
	form := &createArticleForm{}
	if err := ctx.Bind(form); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid form"})
		logger.ErrorWithCTX(ctx, "binding error", err)
		return
	}

	imageCount := len(form.ImageHeaders)
	collectionCount := uint8(len(form.Article.CollectionInfos))
	collectionUUIDs := make([]binaryuuid.UUID, len(form.Article.CollectionInfos))
	orders := make([]uint8, len(form.Article.CollectionInfos))
	for i, collectionInfo := range form.Article.CollectionInfos {
		if *collectionInfo.Order > collectionCount { // uint8, so no need to check sign of number
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request, order is invalid"})
			logger.ErrorWithCTX(ctx, "order invalid", ErrInvalidOrder)
			return
		}
		cuid := binaryuuid.MustParse(collectionInfo.UUID)
		collectionUUIDs[i] = cuid
		orders[i] = *collectionInfo.Order
	}

	imageUUIDs := make([]binaryuuid.UUID, imageCount)
	for i := 0; i < imageCount; i++ {
		buid, err := binaryuuid.NewRandom()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			logger.ErrorWithCTX(ctx, "create image uuid", err)
			return
		}
		imageUUIDs[i] = buid
	}

	// no check uuids duplicated, since uuidv4 duplicate probability is very low, err when insert to DB with unique key

	// check collection is owned
	userId, err := a.d.GetUserIdByUUID(claims.UUID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		logger.ErrorWithCTX(ctx, "get userid by uuid", err)
		return
	}
	if err = a.d.IsCollectionOwned(userId, &collectionUUIDs); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
		logger.ErrorWithCTX(ctx, "bad request", err)
		return
	}

	// upload image first, for consistency, if database write success and imag write file, need to rollback but rollback can be also failed. Then its hard to track and recover
	if err := a.uploadImagesWithUUID(ctx, &imageUUIDs, &form.ImageHeaders); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		logger.ErrorWithCTX(ctx, "upload image", err)
		return
	}

	if err := a.d.CreateNewArticle(userId, form.Article.Title, form.Article.Content, &collectionUUIDs, &imageUUIDs, &orders); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		logger.ErrorWithCTX(ctx, "create new article", err)
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "ok"})
}

type getArticleLinksUri struct {
	Username string `uri:"username" binding:"required"`
}

type getArticleLinksForm struct {
	Offset int `form:"offset,default=0" binding:"min=0"`
	Limit  int `form:"limit,default=20" binding:"min=1,max=20"`
}

func (a *ArticleAPI) GetUserArticleLinksHandler(ctx *gin.Context) {
	form := &getArticleLinksForm{}
	if err := ctx.Bind(form); err != nil {
		logger.ErrorWithCTX(ctx, "bind form", err)
		return
	}

	uri := &getArticleLinksUri{}
	if err := ctx.BindUri(uri); err != nil {
		logger.ErrorWithCTX(ctx, "bind uri", err)
		return
	}

	userId, err := a.d.GetUserIdByName(uri.Username)
	if err != nil {
		ctx.Status(http.StatusNotFound)
		logger.ErrorWithCTX(ctx, "get userid by uuid", err)
		return
	}

	articles, err := a.d.GetArticleLinkIdsByUserId(userId, form.Offset, form.Limit)
	if err != nil {
		ctx.Status(http.StatusNotFound)
		logger.ErrorWithCTX(ctx, "query linkids by user id", err)
		return
	}

	links := make([]string, len(*articles))
	for i, article := range *articles {
		links[i] = base64.URLEncoding.EncodeToString(article[:])
	}

	ctx.JSON(http.StatusOK, gin.H{"links": links})
}

func (a *ArticleAPI) GetArticleHandler(ctx *gin.Context) {
	link := ctx.Param("link")
	linkBytes, err := base64.URLEncoding.DecodeString(link)
	if err != nil {
		ctx.Status(http.StatusNotFound)
		logger.ErrorWithCTX(ctx, "decode link", nil)
		return
	}

	if len(linkBytes) != 16 {
		ctx.Status(http.StatusNotFound)
		logger.ErrorWithCTX(ctx, "link bytes error", nil)
		return
	}

	linkId, err := binaryuuid.FromBytes(linkBytes)
	if err != nil {
		ctx.Status(http.StatusNotFound)
		logger.ErrorWithCTX(ctx, "parse link id", err)
		return
	}

	article, err := a.d.GetArticle(linkId)
	if err != nil {
		ctx.Status(http.StatusNotFound)
		logger.ErrorWithCTX(ctx, "get article", err)
		return
	}
	ctx.JSON(http.StatusOK, article)
}

type getArticleImageHandlerUri struct {
	ImageUUID string `uri:"uuid" binding:"required,uuid"`
}

func (a *ArticleAPI) GetArticleImageHandler(ctx *gin.Context) {
	uri := &getArticleImageHandlerUri{}
	if err := ctx.BindUri(uri); err != nil {
		ctx.Status(http.StatusBadRequest)
		logger.ErrorWithCTX(ctx, "binding uri", err)
		return
	}

	claimsPtr, isExist := ctx.Get("claims")
	var claimerUUID *binaryuuid.UUID
	if isExist {
		claimerUUID = &(claimsPtr.(*auth.AuthClaims)).UUID
	}
	imageUUID := binaryuuid.MustParse(uri.ImageUUID)
	if err := a.d.HasAccessPermissionArticleImage(claimerUUID, imageUUID); err != nil {
		ctx.Status(http.StatusNotFound)
		logger.ErrorWithCTX(ctx, "check permission", err)
		return
	}
	imageBytes, err := a.Storage.GetArticleJPG(ctx, imageUUID)
	if err != nil {
		ctx.Status(http.StatusNotFound)
		logger.ErrorWithCTX(ctx, "get jpg", err)
		return
	}
	ctx.Data(http.StatusOK, "image/jpeg", *imageBytes)
}
