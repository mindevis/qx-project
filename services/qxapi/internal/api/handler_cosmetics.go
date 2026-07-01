package api

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/cosmetics"
)

type CosmeticsHandler struct {
	Service *cosmetics.Service
}

type equipCosmeticsBody struct {
	SkinModel string `json:"skin_model"`
	CapeType  string `json:"cape_type"`
}

func (h *CosmeticsHandler) GetMine(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.Get(c.Request.Context(), owner.UserID)
	if err != nil {
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *CosmeticsHandler) Equip(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req equipCosmeticsBody
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	in := cosmetics.EquipInput{}
	if req.SkinModel != "" {
		in.SkinModel = &req.SkinModel
	}
	if req.CapeType != "" {
		in.CapeType = &req.CapeType
	}
	view, err := h.Service.Equip(c.Request.Context(), owner.UserID, in)
	if err != nil {
		if errors.Is(err, cosmetics.ErrValidation) {
			JSONValidation(c, "invalid cosmetics selection")
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *CosmeticsHandler) UploadSkin(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	data, err := readCosmeticsUpload(c, "skin")
	if err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.UploadSkin(c.Request.Context(), owner.UserID, data)
	if err != nil {
		if isCosmeticsValidation(err) {
			JSONValidation(c, err.Error())
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *CosmeticsHandler) DeleteSkin(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.DeleteSkin(c.Request.Context(), owner.UserID)
	if err != nil {
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *CosmeticsHandler) ListSkinCatalog(c *gin.Context) {
	category := c.Query("category")
	items := cosmetics.ListSkinCatalog(category)
	c.JSON(http.StatusOK, gin.H{"items": items})
}

type applySkinBody struct {
	CatalogID string `json:"catalog_id"`
	Username  string `json:"username"`
}

func (h *CosmeticsHandler) ApplySkin(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req applySkinBody
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	var view *cosmetics.View
	var err error
	switch {
	case strings.TrimSpace(req.CatalogID) != "":
		view, err = h.Service.ApplySkinFromCatalog(c.Request.Context(), owner.UserID, req.CatalogID)
	case strings.TrimSpace(req.Username) != "":
		view, err = h.Service.ApplySkinFromMojang(c.Request.Context(), owner.UserID, req.Username)
	default:
		JSONValidation(c, "catalog_id or username required")
		return
	}
	if err != nil {
		if errors.Is(err, cosmetics.ErrPlayerNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "minecraft player not found")
			return
		}
		if errors.Is(err, cosmetics.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "catalog entry not found")
			return
		}
		if isCosmeticsValidation(err) {
			JSONValidation(c, err.Error())
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *CosmeticsHandler) UploadCape(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	data, err := readCosmeticsUpload(c, "cape")
	if err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.UploadCape(c.Request.Context(), owner.UserID, data)
	if err != nil {
		if isCosmeticsValidation(err) {
			JSONValidation(c, err.Error())
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *CosmeticsHandler) DeleteCape(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.DeleteCape(c.Request.Context(), owner.UserID)
	if err != nil {
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *CosmeticsHandler) ServeCape(c *gin.Context) {
	id := strings.TrimSuffix(c.Param("userId"), ".png")
	ownerID, err := h.Service.ResolveTextureOwner(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, cosmetics.ErrNotFound) {
			c.Status(http.StatusNotFound)
			return
		}
		JSONInternal(c)
		return
	}
	data, err := h.Service.ReadCapePNG(ownerID)
	if err != nil {
		if errors.Is(err, cosmetics.ErrNoCape) {
			c.Status(http.StatusNotFound)
			return
		}
		JSONInternal(c)
		return
	}
	c.Data(http.StatusOK, "image/png", data)
}

func (h *CosmeticsHandler) ServeSkin(c *gin.Context) {
	id := strings.TrimSuffix(c.Param("userId"), ".png")
	ownerID, err := h.Service.ResolveTextureOwner(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, cosmetics.ErrNotFound) {
			c.Status(http.StatusNotFound)
			return
		}
		JSONInternal(c)
		return
	}
	data, err := h.Service.ReadSkinPNG(ownerID)
	if err != nil {
		if errors.Is(err, cosmetics.ErrNoSkin) {
			c.Status(http.StatusNotFound)
			return
		}
		JSONInternal(c)
		return
	}
	c.Data(http.StatusOK, "image/png", data)
}

func readCosmeticsUpload(c *gin.Context, field string) ([]byte, error) {
	file, err := c.FormFile(field)
	if err != nil {
		return nil, errors.New("missing " + field + " file")
	}
	if file.Size > 256*1024 {
		return nil, errors.New("file too large")
	}
	f, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, 256*1024))
}

func isCosmeticsValidation(err error) bool {
	return errors.Is(err, cosmetics.ErrInvalidSkinFormat) ||
		errors.Is(err, cosmetics.ErrInvalidSkinSize) ||
		errors.Is(err, cosmetics.ErrSkinTooLarge)
}
