// Package grpchandler provides gRPC implementations of the application services.
package grpchandler

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/semmidev/go-todo-app/gen/todo/v1"
	"github.com/semmidev/go-todo-app/internal/adapter/driving/grpc/grpcerr"
	"github.com/semmidev/go-todo-app/internal/adapter/driving/grpc/interceptor"
	"github.com/semmidev/go-todo-app/internal/common/filter"
	"github.com/semmidev/go-todo-app/internal/common/validation"
	"github.com/semmidev/go-todo-app/internal/common/wideevent"
	"github.com/semmidev/go-todo-app/internal/domain/todo"
	"github.com/semmidev/go-todo-app/internal/port/input"
)

// ─── TagServer ────────────────────────────────────────────────────────────────

// TagServer handles tag-related operations via gRPC.
// It depends on the input.TagUseCase interface, NOT the concrete service.
type TagServer struct {
	pb.UnimplementedTagServiceServer
	svc       input.TagUseCase
	validator *validation.Validator
}

// NewTagServer creates a new TagServer.
func NewTagServer(svc input.TagUseCase, validator *validation.Validator) *TagServer {
	return &TagServer{svc: svc, validator: validator}
}

// CreateTag creates a new tag for the authenticated user.
func (s *TagServer) CreateTag(ctx context.Context, req *pb.CreateTagRequest) (*pb.Tag, error) {
	u, ok := interceptor.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}
	params := input.CreateTagParams{
		UserID: u.ID(), Name: req.GetName(), Color: req.GetColor(),
	}
	if err := s.validator.Struct(params); err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	t, err := s.svc.CreateTag(ctx, params)
	if err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	return tagToProto(t), nil
}

// GetTag retrieves a specific tag by its ID.
func (s *TagServer) GetTag(ctx context.Context, req *pb.GetTagRequest) (*pb.Tag, error) {
	u, ok := interceptor.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}
	tagID, err := uuid.Parse(req.GetTagId())
	if err != nil {
		return nil, grpcerr.NewInvalidArgument("invalid tag_id")
	}
	params := input.GetTagParams{TagID: tagID, UserID: u.ID()}
	if err := s.validator.Struct(params); err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	t, err := s.svc.GetTag(ctx, params)
	if err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	return tagToProto(t), nil
}

// UpdateTag updates an existing tag.
func (s *TagServer) UpdateTag(ctx context.Context, req *pb.UpdateTagRequest) (*pb.Tag, error) {
	u, ok := interceptor.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}
	tagID, err := uuid.Parse(req.GetTagId())
	if err != nil {
		return nil, grpcerr.NewInvalidArgument("invalid tag_id")
	}
	params := input.UpdateTagParams{
		TagID: tagID, UserID: u.ID(), Name: req.GetName(), Color: req.GetColor(),
	}
	if err := s.validator.Struct(params); err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	t, err := s.svc.UpdateTag(ctx, params)
	if err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	return tagToProto(t), nil
}

// DeleteTag removes a tag.
func (s *TagServer) DeleteTag(ctx context.Context, req *pb.DeleteTagRequest) (*emptypb.Empty, error) {
	u, ok := interceptor.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}
	tagID, err := uuid.Parse(req.GetTagId())
	if err != nil {
		return nil, grpcerr.NewInvalidArgument("invalid tag_id")
	}
	params := input.DeleteTagParams{TagID: tagID, UserID: u.ID()}
	if err := s.validator.Struct(params); err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	if err := s.svc.DeleteTag(ctx, params); err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	return &emptypb.Empty{}, nil
}

// ListTags returns all tags belonging to the authenticated user.
func (s *TagServer) ListTags(ctx context.Context, _ *emptypb.Empty) (*pb.ListTagsResponse, error) {
	u, ok := interceptor.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}
	params := input.ListTagsParams{UserID: u.ID()}
	if err := s.validator.Struct(params); err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	tags, err := s.svc.ListTags(ctx, params)
	if err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	pbTags := make([]*pb.Tag, len(tags))
	for i, t := range tags {
		pbTags[i] = tagToProto(t)
	}
	return &pb.ListTagsResponse{Tags: pbTags, Total: int32(len(pbTags))}, nil
}

var _ pb.TagServiceServer = (*TagServer)(nil)

// ─── TodoServer ───────────────────────────────────────────────────────────────

// TodoServer handles todo-related operations via gRPC.
// It depends on the input.TodoUseCase interface, NOT the concrete service.
type TodoServer struct {
	pb.UnimplementedTodoServiceServer
	svc       input.TodoUseCase
	validator *validation.Validator
}

// NewTodoServer creates a new TodoServer.
func NewTodoServer(svc input.TodoUseCase, validator *validation.Validator) *TodoServer {
	return &TodoServer{svc: svc, validator: validator}
}

// CreateTodo creates a new todo item.
func (s *TodoServer) CreateTodo(ctx context.Context, req *pb.CreateTodoRequest) (*pb.Todo, error) {
	u, ok := interceptor.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}
	tagIDs, err := parseUUIDs(req.GetTagIds())
	if err != nil {
		return nil, grpcerr.NewInvalidArgument("invalid tag_ids")
	}
	var dueDate *time.Time
	if req.GetDueDate() != nil {
		t := req.GetDueDate().AsTime()
		dueDate = &t
	}
	params := input.CreateTodoParams{
		UserID: u.ID(), Title: req.GetTitle(), Description: req.GetDescription(),
		Priority: protoPriorityToDomain(req.GetPriority()), DueDate: dueDate, TagIDs: tagIDs,
		Reminders: req.GetReminders(),
	}
	if err := s.validator.Struct(params); err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	// Enrich wide event with business context
	wideevent.Add(ctx,
		slog.String("todo_title", params.Title),
		slog.String("todo_priority", string(params.Priority)),
		slog.Int("tag_count", len(params.TagIDs)),
	)
	t, err := s.svc.CreateTodo(ctx, params)
	if err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	wideevent.Add(ctx, slog.String("todo_id", t.ID().String()))
	return todoToProto(t), nil
}

// GetTodo retrieves a specific todo item.
func (s *TodoServer) GetTodo(ctx context.Context, req *pb.GetTodoRequest) (*pb.Todo, error) {
	u, ok := interceptor.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}
	todoID, err := uuid.Parse(req.GetTodoId())
	if err != nil {
		return nil, grpcerr.NewInvalidArgument("invalid todo_id")
	}
	params := input.GetTodoParams{TodoID: todoID, UserID: u.ID()}
	if err := s.validator.Struct(params); err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	wideevent.Add(ctx, slog.String("todo_id", todoID.String()))
	t, err := s.svc.GetTodo(ctx, params)
	if err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	return todoToProto(t), nil
}

// UpdateTodo updates an existing todo item.
func (s *TodoServer) UpdateTodo(ctx context.Context, req *pb.UpdateTodoRequest) (*pb.Todo, error) {
	u, ok := interceptor.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}
	todoID, err := uuid.Parse(req.GetTodoId())
	if err != nil {
		return nil, grpcerr.NewInvalidArgument("invalid todo_id")
	}
	tagIDs, err := parseUUIDs(req.GetTagIds())
	if err != nil {
		return nil, grpcerr.NewInvalidArgument("invalid tag_ids")
	}
	var dueDate *time.Time
	if req.GetDueDate() != nil {
		t := req.GetDueDate().AsTime()
		dueDate = &t
	}
	params := input.UpdateTodoParams{
		TodoID: todoID, UserID: u.ID(), Title: req.GetTitle(), Description: req.GetDescription(),
		Status: protoStatusToDomain(req.GetStatus()), Priority: protoPriorityToDomain(req.GetPriority()),
		DueDate: dueDate, TagIDs: tagIDs,
		Reminders: req.GetReminders(),
	}
	if err := s.validator.Struct(params); err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	// Enrich wide event with update context
	wideevent.Add(ctx,
		slog.String("todo_id", todoID.String()),
		slog.String("todo_status", string(params.Status)),
	)
	t, err := s.svc.UpdateTodo(ctx, params)
	if err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	return todoToProto(t), nil
}

// UpdateTodoStatus updates only the status of a todo item.
func (s *TodoServer) UpdateTodoStatus(ctx context.Context, req *pb.UpdateTodoStatusRequest) (*pb.Todo, error) {
	u, ok := interceptor.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}
	todoID, err := uuid.Parse(req.GetTodoId())
	if err != nil {
		return nil, grpcerr.NewInvalidArgument("invalid todo_id")
	}
	params := input.UpdateTodoStatusParams{
		TodoID: todoID, UserID: u.ID(),
		Status: protoStatusToDomain(req.GetStatus()),
	}
	if err := s.validator.Struct(params); err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	// Enrich wide event with update context
	wideevent.Add(ctx,
		slog.String("todo_id", todoID.String()),
		slog.String("todo_status", string(params.Status)),
	)
	t, err := s.svc.UpdateTodoStatus(ctx, params)
	if err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	return todoToProto(t), nil
}

// DeleteTodo removes a todo item.
func (s *TodoServer) DeleteTodo(ctx context.Context, req *pb.DeleteTodoRequest) (*emptypb.Empty, error) {
	u, ok := interceptor.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}
	todoID, err := uuid.Parse(req.GetTodoId())
	if err != nil {
		return nil, grpcerr.NewInvalidArgument("invalid todo_id")
	}
	params := input.DeleteTodoParams{TodoID: todoID, UserID: u.ID()}
	if err := s.validator.Struct(params); err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	wideevent.Add(ctx, slog.String("todo_id", todoID.String()))
	if err := s.svc.DeleteTodo(ctx, params); err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	return &emptypb.Empty{}, nil
}

// ListTodos returns a paginated list of todos for the authenticated user.
func (s *TodoServer) ListTodos(ctx context.Context, req *pb.ListTodosRequest) (*pb.ListTodosResponse, error) {
	u, ok := interceptor.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}
	var statusFilter *todo.Status
	if req.GetStatus() != pb.TodoStatus_TODO_STATUS_UNSPECIFIED {
		st := protoStatusToDomain(req.GetStatus())
		statusFilter = &st
	}
	var tagIDFilter *uuid.UUID
	if req.GetTagId() != "" {
		tagID, err := uuid.Parse(req.GetTagId())
		if err != nil {
			return nil, grpcerr.NewInvalidArgument("invalid tag_id")
		}
		tagIDFilter = &tagID
	}

	// Build from canonical Filter defaults, then override from request
	f := filter.NewFilter()
	if p := int(req.GetPage()); p > 0 {
		f.CurrentPage = p
	}
	if ps := int(req.GetPageSize()); ps > 0 {
		f.PerPage = ps
	}
	if kw := req.GetKeyword(); kw != "" {
		f.Keyword = kw
	}
	if sb := req.GetSortBy(); sb != "" {
		f.SortBy = sb
	}
	if sd := req.GetSortDirection(); sd != "" {
		f.SortDirection = sd
	}
	if req.GetStartDate() != nil {
		t := req.GetStartDate().AsTime()
		f.StartDate = &t
	}
	if req.GetEndDate() != nil {
		t := req.GetEndDate().AsTime()
		f.EndDate = &t
	}

	params := input.ListTodosParams{
		Filter: f,
		UserID: u.ID(),
		Status: statusFilter,
		TagID:  tagIDFilter,
	}
	if err := s.validator.Struct(params); err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	// Enrich wide event with query context
	wideevent.Add(ctx,
		slog.Int("current_page", params.CurrentPage),
		slog.Int("per_page", params.PerPage),
		slog.String("keyword", params.Keyword),
	)
	result, err := s.svc.ListTodos(ctx, params)
	if err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	pbTodos := make([]*pb.Todo, len(result.Todos))
	for i, t := range result.Todos {
		pbTodos[i] = todoToProto(t)
	}
	wideevent.Add(ctx, slog.Int("result_count", len(pbTodos)))
	return &pb.ListTodosResponse{
		Todos:  pbTodos,
		Paging: pagingToProto(result.Paging),
	}, nil
}

// AddTagToTodo associates a tag with a todo item.
func (s *TodoServer) AddTagToTodo(ctx context.Context, req *pb.AddTagToTodoRequest) (*pb.Todo, error) {
	u, ok := interceptor.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}
	todoID, err := uuid.Parse(req.GetTodoId())
	if err != nil {
		return nil, grpcerr.NewInvalidArgument("invalid todo_id")
	}
	tagID, err := uuid.Parse(req.GetTagId())
	if err != nil {
		return nil, grpcerr.NewInvalidArgument("invalid tag_id")
	}
	params := input.AddTagToTodoParams{TodoID: todoID, TagID: tagID, UserID: u.ID()}
	if err := s.validator.Struct(params); err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	t, err := s.svc.AddTagToTodo(ctx, params)
	if err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	return todoToProto(t), nil
}

// RemoveTagFromTodo disassociates a tag from a todo item.
func (s *TodoServer) RemoveTagFromTodo(ctx context.Context, req *pb.RemoveTagFromTodoRequest) (*pb.Todo, error) {
	u, ok := interceptor.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}
	todoID, err := uuid.Parse(req.GetTodoId())
	if err != nil {
		return nil, grpcerr.NewInvalidArgument("invalid todo_id")
	}
	tagID, err := uuid.Parse(req.GetTagId())
	if err != nil {
		return nil, grpcerr.NewInvalidArgument("invalid tag_id")
	}
	params := input.RemoveTagFromTodoParams{TodoID: todoID, TagID: tagID, UserID: u.ID()}
	if err := s.validator.Struct(params); err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	t, err := s.svc.RemoveTagFromTodo(ctx, params)
	if err != nil {
		return nil, grpcerr.FromAppError(ctx, err)
	}
	return todoToProto(t), nil
}

var _ pb.TodoServiceServer = (*TodoServer)(nil)

// ── Paging helper ─────────────────────────────────────────────────────────────

// pagingToProto converts a domain pagination model to its gRPC representation.
func pagingToProto(p *filter.Paging) *pb.Paging {
	if p == nil {
		return &pb.Paging{}
	}
	return &pb.Paging{
		HasPreviousPage:        p.HasPreviousPage,
		HasNextPage:            p.HasNextPage,
		CurrentPage:            int32(p.CurrentPage),
		PerPage:                int32(p.PerPage),
		TotalData:              int32(p.TotalData),
		TotalDataInCurrentPage: int32(p.TotalDataInCurrentPage),
		LastPage:               int32(p.LastPage),
		From:                   int32(p.From),
		To:                     int32(p.To),
	}
}
