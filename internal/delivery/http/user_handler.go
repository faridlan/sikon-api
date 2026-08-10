package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type UserHandler interface {
	Register(c *fiber.Ctx) error
	GetProfile(c *fiber.Ctx) error
	ListUsers(c *fiber.Ctx) error
	GetPublicSalesList(c *fiber.Ctx) error
	UpdateUser(c *fiber.Ctx) error
	DeleteUser(c *fiber.Ctx) error
}

type userHandler struct {
	userUsecase domain.UserUsecase
}

func NewUserHandler(uu domain.UserUsecase) UserHandler {
	return &userHandler{
		userUsecase: uu,
	}
}

// @Summary Register a new user
// @Description Mendaftarkan pengguna baru (Admin/Sales) ke dalam sistem SIKOn
// @Tags Users
// @Accept json
// @Produce json
// @Param request body dto.UserRegisterRequest true "Data Registrasi User"
// @Success 201 {object} utils.SuccessResponse[dto.UserResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 409 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /users/register [post]
func (h *userHandler) Register(c *fiber.Ctx) error {
	var req dto.UserRegisterRequest

	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}

	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	domainReq := domain.UserRegisterInput{
		Name:       req.Name,
		Email:      req.Email,
		Password:   req.Password,
		Role:       domain.Role(req.Role),
		ImageURL:   req.ImageURL,
		Phone:      req.Phone,
		StatusText: req.StatusText,
		IsActive:   isActive,
		SortOrder:  req.SortOrder,
	}

	user, err := h.userUsecase.Register(c.Context(), domainReq)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil mendaftarkan user", dto.ToUserResponse(user))
}

// @Summary Get User Profile
// @Description Mengambil detail profil user berdasarkan ID
// @Tags Users
// @Produce json
// @Param id path string true "User ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[dto.UserResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /users/{id} [get]
func (h *userHandler) GetProfile(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	user, err := h.userUsecase.GetProfile(c.Context(), id)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil profil user", dto.ToUserResponse(user))
}

// @Summary List Users
// @Description Mengambil daftar seluruh user dengan pagination
// @Tags Users
// @Produce json
// @Param page query int false "Nomor Halaman" default(1)
// @Param limit query int false "Batas Data per Halaman" default(10)
// @Param search query string false "Pencarian berdasarkan nama atau email"
// @Param role query string false "Filter berdasarkan role (admin/sales)"
// @Success 200 {object} utils.PaginatedResponse[dto.UserResponse]
// @Failure 500 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /users [get]
func (h *userHandler) ListUsers(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	query := domain.PaginationQuery{
		Page:  page,
		Limit: limit,
	}

	filter := domain.UserFilter{
		Search: c.Query("search", ""),
		Role:   c.Query("role", ""),
	}

	users, meta, err := h.userUsecase.ListUsers(c.Context(), query, filter)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccessPaginated(c, "Berhasil mengambil daftar user", dto.ToUserResponseList(users), meta)
}

// @Summary Get Public Sales Marketing Directory
// @Description Mengambil daftar tim marketing aktif untuk landing page / widget WhatsApp
// @Tags Public
// @Produce json
// @Success 200 {object} utils.SuccessResponse[[]dto.PublicSalesResponse]
// @Failure 500 {object} utils.ErrorResponse
// @Router /users/public/sales [get]
func (h *userHandler) GetPublicSalesList(c *fiber.Ctx) error {
	salesList, err := h.userUsecase.GetPublicSalesList(c.Context())
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil daftar tim marketing", dto.ToPublicSalesResponseList(salesList))
}

// @Summary Update User
// @Description Memperbarui nama, role, foto, atau informasi WhatsApp marketing dari user
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID (UUID)"
// @Param request body dto.UserUpdateRequest true "Data Update User"
// @Success 200 {object} utils.SuccessResponse[dto.UserResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /users/{id} [put]
func (h *userHandler) UpdateUser(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var req dto.UserUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}

	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	domainReq := domain.UserUpdateInput{
		Name:       req.Name,
		Role:       domain.Role(req.Role),
		ImageURL:   req.ImageURL,
		Phone:      req.Phone,
		StatusText: req.StatusText,
		IsActive:   req.IsActive,
		SortOrder:  req.SortOrder,
	}

	user, err := h.userUsecase.UpdateUser(c.Context(), id, domainReq)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil memperbarui data user", dto.ToUserResponse(user))
}

// @Summary Delete User
// @Description Menghapus data user secara permanen
// @Tags Users
// @Produce json
// @Param id path string true "User ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /users/{id} [delete]
func (h *userHandler) DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.userUsecase.DeleteUser(c.Context(), id); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menghapus user", nil)
}
