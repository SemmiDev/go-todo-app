package grpchandler

import (
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/semmidev/go-todo-app/gen/todo/v1"
	"github.com/semmidev/go-todo-app/internal/domain/todo"
)

// ─── Proto ↔ Domain mappers ──────────────────────────────────────────────────

func tagToProto(t *todo.Tag) *pb.Tag {
	return &pb.Tag{
		Id:        t.ID().String(),
		UserId:    t.UserID().String(),
		Name:      t.Name(),
		Color:     t.Color(),
		CreatedAt: timestamppb.New(t.CreatedAt()),
		UpdatedAt: timestamppb.New(t.UpdatedAt()),
	}
}

func todoToProto(t *todo.Todo) *pb.Todo {
	pbTags := make([]*pb.Tag, len(t.Tags()))
	for i, tag := range t.Tags() {
		pbTags[i] = tagToProto(tag)
	}
	out := &pb.Todo{
		Id:          t.ID().String(),
		UserId:      t.UserID().String(),
		Title:       t.Title(),
		Description: t.Description(),
		Status:      domainStatusToProto(t.Status()),
		Priority:    domainPriorityToProto(t.Priority()),
		CreatedAt:   timestamppb.New(t.CreatedAt()),
		UpdatedAt:   timestamppb.New(t.UpdatedAt()),
		Tags:        pbTags,
	}
	if t.DueDate() != nil {
		out.DueDate = timestamppb.New(*t.DueDate())
	}
	return out
}

func protoStatusToDomain(s pb.TodoStatus) todo.Status {
	switch s {
	case pb.TodoStatus_TODO_STATUS_IN_PROGRESS:
		return todo.StatusInProgress
	case pb.TodoStatus_TODO_STATUS_DONE:
		return todo.StatusDone
	default:
		return todo.StatusPending
	}
}

func domainStatusToProto(s todo.Status) pb.TodoStatus {
	switch s {
	case todo.StatusInProgress:
		return pb.TodoStatus_TODO_STATUS_IN_PROGRESS
	case todo.StatusDone:
		return pb.TodoStatus_TODO_STATUS_DONE
	default:
		return pb.TodoStatus_TODO_STATUS_PENDING
	}
}

func protoPriorityToDomain(p pb.TodoPriority) todo.Priority {
	switch p {
	case pb.TodoPriority_TODO_PRIORITY_LOW:
		return todo.PriorityLow
	case pb.TodoPriority_TODO_PRIORITY_HIGH:
		return todo.PriorityHigh
	default:
		return todo.PriorityMedium
	}
}

func domainPriorityToProto(p todo.Priority) pb.TodoPriority {
	switch p {
	case todo.PriorityLow:
		return pb.TodoPriority_TODO_PRIORITY_LOW
	case todo.PriorityHigh:
		return pb.TodoPriority_TODO_PRIORITY_HIGH
	default:
		return pb.TodoPriority_TODO_PRIORITY_MEDIUM
	}
}

func parseUUIDs(ids []string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		result = append(result, parsed)
	}
	return result, nil
}
