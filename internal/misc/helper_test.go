package misc

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"sarabi/internal/types"
	"testing"
)

func TestParseContainerName(t *testing.T) {
	tests := []struct {
		name        string
		args        string
		expectedErr error
		expected    *types.ContainerIdentity
	}{
		{
			name: "parse deployment container name",
			args: "582ef9e6ec2d452bb0816dc704cb29a3-s-0",
			expected: &types.ContainerIdentity{
				DeploymentID: uuid.MustParse("582ef9e6-ec2d-452b-b081-6dc704cb29a3"),
				Environment:  "s",
				InstanceID:   0,
			},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseContainerIdentity("", test.name)
			assert.Equal(t, test.expected, got)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
