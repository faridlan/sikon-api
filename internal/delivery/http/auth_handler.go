package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type AuthHandler interface {
	Login(c *fiber.Ctx) error
}

type authHandler struct {
	authUsecase domain.AuthUsecase
}

func NewAuthHandler(au domain.AuthUsecase) AuthHandler {
	return &authHandler{
		authUsecase: au,
	}
}

// @Summary Login User
// @Description Autentikasi user dan mendapatkan JWT Token untuk hak akses sistem SIKOn
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Credentials Login"
// @Success 200 {object} utils.SuccessResponse[dto.AuthResponseDTO]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 403 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /auth/login [post]
func (h *authHandler) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest

	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}

	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	domainReq := domain.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	}

	res, err := h.authUsecase.Login(c.Context(), domainReq)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Login berhasil", dto.ToAuthResponseDTO(res))
}
