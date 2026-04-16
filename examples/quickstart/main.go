package main

import (
	"errors"
	"fmt"

	"github.com/Deimvis-go/valid"
)

// User is validated in two ways:
//   - via struct tags (Name must be non-empty) handled by
//     github.com/go-playground/validator/v10
//   - via a ValidateSelf method (Age must be non-negative) handled by valid.
type User struct {
	Name string `validate:"required"`
	Age  int
}

func (u *User) ValidateSelf() error {
	if u.Age < 0 {
		return errors.New("age must be non-negative")
	}
	return nil
}

// Team also implements ValidateSelf. Note that valid.Deep will automatically
// recurse into Team.Members and validate each User, so Team does not need to
// re-implement that logic.
type Team struct {
	Name    string `validate:"required"`
	Members []*User
}

func (t *Team) ValidateSelf() error {
	if len(t.Members) == 0 {
		return errors.New("team must have at least one member")
	}
	return nil
}

func main() {
	team := &Team{
		Name: "example",
		Members: []*User{
			{Name: "alice", Age: 30},
			{Name: "bob", Age: -1}, // invalid
		},
	}
	fmt.Println(valid.Deep(team))
}
