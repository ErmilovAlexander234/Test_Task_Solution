package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	taskdomain "example.com/taskservice/internal/domain/task"
	taskusecase "example.com/taskservice/internal/usecase/task"
)

type TaskHandler struct {
	usecase taskusecase.Usecase
}

func NewTaskHandler(usecase taskusecase.Usecase) *TaskHandler {
	return &TaskHandler{usecase: usecase}
}

//---------жесть ты круд-----------------------------------

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req taskMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	created, err := h.usecase.Create(r.Context(), taskusecase.CreateInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, newTaskDTO(created))
}

func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	task, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		writeUsecaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newTaskDTO(task))
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var req taskMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	updated, err := h.usecase.Update(r.Context(), id, taskusecase.UpdateInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newTaskDTO(updated))
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.usecase.Delete(r.Context(), id); err != nil {
		writeUsecaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.usecase.List(r.Context())
	if err != nil {
		writeUsecaseError(w, err)
		return
	}
	response := make([]taskDTO, len(tasks))
	for i := range tasks {
		response[i] = newTaskDTO(&tasks[i])
	}
	writeJSON(w, http.StatusOK, response)
}

//-----------------------повтор тасок------------------------------------

func (h *TaskHandler) CreateRecurring(w http.ResponseWriter, r *http.Request) {
	var req createRecurringRequestDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	rule, err := dtoToRecurrenceRule(req.Recurrence)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	tasks, err := h.usecase.CreateRecurring(r.Context(), taskusecase.CreateRecurringInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		Recurrence:  rule,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}
	response := make([]taskDTO, len(tasks))
	for i := range tasks {
		response[i] = newTaskDTO(&tasks[i])
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *TaskHandler) GetRecurrenceRule(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	rule, err := h.usecase.GetRecurrenceRule(r.Context(), id)
	if err != nil {
		writeUsecaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newRecurrenceRuleResponseDTO(rule))
}

func (h *TaskHandler) ListRecurrenceRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.usecase.ListRecurrenceRules(r.Context())
	if err != nil {
		writeUsecaseError(w, err)
		return
	}
	response := make([]recurrenceRuleResponseDTO, len(rules))
	for i := range rules {
		response[i] = newRecurrenceRuleResponseDTO(&rules[i])
	}
	writeJSON(w, http.StatusOK, response)
}

// --------------pounce к helper----------------------------------

func dtoToRecurrenceRule(dto recurrenceRuleDTO) (taskdomain.RecurrenceRule, error) {
	rule := taskdomain.RecurrenceRule{
		Type:        taskdomain.RecurrenceType(dto.Type),
		EveryNDays:  dto.EveryNDays,
		MonthlyDays: dto.MonthlyDays,
	}
	if dto.StartDate != nil {
		d, err := time.Parse("2006-01-02", *dto.StartDate)
		if err != nil {
			return taskdomain.RecurrenceRule{}, fmt.Errorf("invalid start_date %q: expected YYYY-MM-DD", *dto.StartDate)
		}
		d = d.UTC()
		rule.StartDate = &d
	}
	for _, s := range dto.SpecificDates {
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			return taskdomain.RecurrenceRule{}, fmt.Errorf("invalid date %q in specific_dates: expected YYYY-MM-DD", s)
		}
		rule.SpecificDates = append(rule.SpecificDates, d.UTC())
	}
	if dto.EvenOdd != nil {
		eo := taskdomain.EvenOdd(*dto.EvenOdd)
		rule.EvenOdd = &eo
	}
	return rule, nil
}

//------------------общее-------------------------------------

func getIDFromRequest(r *http.Request) (int64, error) {
	rawID := mux.Vars(r)["id"]
	if rawID == "" {
		return 0, errors.New("missing id")
	}
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func writeUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, taskdomain.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, taskusecase.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// ------------------------Delete при мне (повтор)-----------------------------
func (h *TaskHandler) DeleteRuleTasks(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.usecase.DeleteRuleTasks(r.Context(), id); err != nil {
		writeUsecaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) DeleteRecurrenceRule(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.usecase.DeleteRecurrenceRule(r.Context(), id); err != nil {
		writeUsecaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --------возврат-------------
func (h *TaskHandler) ListTasksByRuleID(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	tasks, err := h.usecase.ListTasksByRuleID(r.Context(), id)
	if err != nil {
		writeUsecaseError(w, err)
		return
	}
	response := make([]taskDTO, len(tasks))
	for i := range tasks {
		response[i] = newTaskDTO(&tasks[i])
	}
	writeJSON(w, http.StatusOK, response)
}
