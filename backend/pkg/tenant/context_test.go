package tenant

import (
	"context"
	"testing"
)

func TestWithSpaceID_And_SpaceID(t *testing.T) {
	ctx := context.Background()

	// Not set
	_, ok := SpaceID(ctx)
	if ok {
		t.Error("SpaceID should return false when not set")
	}

	// Set and retrieve
	ctx = WithSpaceID(ctx, 42)
	sid, ok := SpaceID(ctx)
	if !ok || sid != 42 {
		t.Errorf("SpaceID = %d, %v; want 42, true", sid, ok)
	}
}

func TestMustSpaceID_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustSpaceID should panic when space_id not set")
		}
	}()
	MustSpaceID(context.Background())
}

func TestMustSpaceID_Success(t *testing.T) {
	ctx := WithSpaceID(context.Background(), 99)
	sid := MustSpaceID(ctx)
	if sid != 99 {
		t.Errorf("MustSpaceID = %d; want 99", sid)
	}
}

func TestSpaceIDOrDefault(t *testing.T) {
	ctx := context.Background()
	if got := SpaceIDOrDefault(ctx, 7); got != 7 {
		t.Errorf("SpaceIDOrDefault = %d; want 7 (default)", got)
	}

	ctx = WithSpaceID(ctx, 42)
	if got := SpaceIDOrDefault(ctx, 7); got != 42 {
		t.Errorf("SpaceIDOrDefault = %d; want 42", got)
	}
}

func TestRequireSpaceID(t *testing.T) {
	ctx := context.Background()
	_, err := RequireSpaceID(ctx)
	if err == nil {
		t.Error("RequireSpaceID should error when not set")
	}

	ctx = WithSpaceID(ctx, 42)
	sid, err := RequireSpaceID(ctx)
	if err != nil || sid != 42 {
		t.Errorf("RequireSpaceID = %d, %v; want 42, nil", sid, err)
	}
}

func TestWithUserID_And_UserID(t *testing.T) {
	ctx := context.Background()
	_, ok := UserID(ctx)
	if ok {
		t.Error("UserID should return false when not set")
	}

	ctx = WithUserID(ctx, "user-123")
	uid, ok := UserID(ctx)
	if !ok || uid != "user-123" {
		t.Errorf("UserID = %q, %v; want user-123, true", uid, ok)
	}
}
