package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type UploadHandler interface {
	UploadImage(c *fiber.Ctx) error
}

type uploadHandler struct {
	uploadUsecase domain.UploadUsecase
}

func NewUploadHandler(uu domain.UploadUsecase) UploadHandler {
	return &uploadHandler{uploadUsecase: uu}
}

// @Summary Upload Image
// @Tags Uploads
// @Accept multipart/form-data
// @Produce json
// @Param folder query string false "Nama folder tujuan" default(general)
// @Param file formData file true "File gambar (Max 2MB)"
// @Success 200 {object} utils.SuccessResponse[map[string]string]
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /uploads/image [post]
func (h *uploadHandler) UploadImage(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "File gambar tidak ditemukan dalam request")
	}

	folder := c.Query("folder", "general")

	imageURL, err := h.uploadUsecase.UploadImage(c.Context(), file, folder)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengunggah gambar", fiber.Map{
		"url": imageURL,
	})
}
