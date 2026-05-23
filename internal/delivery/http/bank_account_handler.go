package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type BankAccountHandler struct {
	bankAccountUsecase domain.BankAccountUsecase
}

func NewBankAccountHandler(bu domain.BankAccountUsecase) *BankAccountHandler {
	return &BankAccountHandler{
		bankAccountUsecase: bu,
	}
}

// @Summary Create Bank Account
// @Description Menambahkan rekening bank (Perusahaan / Sales)
// @Tags Bank Accounts
// @Accept json
// @Produce json
// @Param request body dto.BankAccountCreateRequest true "Data Rekening Baru"
// @Success 201 {object} utils.SuccessResponse[dto.BankAccountResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /bank-accounts [post]
func (h *BankAccountHandler) CreateAccount(c *fiber.Ctx) error {
	var req dto.BankAccountCreateRequest

	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}

	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	account := &domain.BankAccount{
		UserID:        req.UserID,
		BankName:      req.BankName,
		AccountNumber: req.AccountNumber,
		AccountName:   req.AccountName,
	}

	if err := h.bankAccountUsecase.CreateAccount(c.Context(), account); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil menambahkan rekening", dto.ToBankAccountResponse(account))
}

// @Summary List All Bank Accounts
// @Description Mengambil daftar seluruh rekening bank
// @Tags Bank Accounts
// @Produce json
// @Param page query int false "Nomor Halaman" default(1)
// @Param limit query int false "Batas Data per Halaman" default(10)
// @Success 200 {object} utils.PaginatedResponse[dto.BankAccountResponse]
// @Failure 500 {object} utils.ErrorResponse
// @Router /bank-accounts [get]
func (h *BankAccountHandler) ListAccounts(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	query := domain.PaginationQuery{Page: page, Limit: limit}

	accounts, meta, err := h.bankAccountUsecase.ListAccounts(c.Context(), query)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccessPaginated(c, "Berhasil mengambil daftar rekening", dto.ToBankAccountResponseList(accounts), meta)
}

// @Summary Get Global Bank Accounts
// @Description Mengambil daftar rekening khusus perusahaan (User ID null)
// @Tags Bank Accounts
// @Produce json
// @Success 200 {object} utils.SuccessResponse[[]dto.BankAccountResponse]
// @Failure 500 {object} utils.ErrorResponse
// @Router /bank-accounts/global [get]
func (h *BankAccountHandler) GetGlobalAccounts(c *fiber.Ctx) error {
	accounts, err := h.bankAccountUsecase.GetGlobalAccounts(c.Context())
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil rekening global", dto.ToBankAccountResponseList(accounts))
}

// @Summary Delete Bank Account
// @Description Menghapus rekening secara permanen
// @Tags Bank Accounts
// @Produce json
// @Param id path string true "Bank Account ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /bank-accounts/{id} [delete]
func (h *BankAccountHandler) DeleteAccount(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.bankAccountUsecase.DeleteAccount(c.Context(), id); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menghapus rekening", nil)
}
