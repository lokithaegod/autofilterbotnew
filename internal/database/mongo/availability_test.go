package mongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestAvailabilityIdentityAndPendingGeneration(t *testing.T) {
	base := AvailabilityAlert{UserID: 10, Title: "hunger games", Year: "2023", Audio: "tam"}
	base.ID = base.key()
	base.Generation = primitive.NewObjectID()
	other := base
	other.UserID++
	if base.key() == other.key() {
		t.Fatal("different users share an alert identity")
	}
	other = base
	other.Audio = "hin"
	if base.key() == other.key() {
		t.Fatal("different languages share an alert identity")
	}
	other = base
	other.Generation = primitive.NewObjectID()
	if base.pendingFilter()["generation"] == other.pendingFilter()["generation"] {
		t.Fatal("a cancelled alert could claim a newly registered subscription")
	}
}
