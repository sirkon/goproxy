package errors

import (
	"io"
	"testing"
)

func Test_Wrapping(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "raw-trivial",
			err: &wrappedError{
				msgs: []string{"1"},
				err:  io.EOF,
			},
			want: "1: " + io.EOF.Error(),
		},
		{
			name: "raw-typical",
			err: &wrappedError{
				msgs: []string{"1", "2", "3", "4"},
				err:  io.EOF,
			},
			want: "4: 3: 2: 1: " + io.EOF.Error(),
		},
		{
			name: "wrap",
			err:  Wrapf(Wrap(io.EOF, "1"), "%d", 2),
			want: "2: 1: " + io.EOF.Error(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_wrappedError_Unwrap(t *testing.T) {
	tests := []struct {
		name    string
		err     *wrappedError
		wantErr error
	}{
		{
			name:    "trivial",
			err:     Wrap(io.EOF, "1").(*wrappedError),
			wantErr: io.EOF,
		},
		{
			name:    "generic",
			err:     Wrap(Wrap(io.EOF, "1"), "2").(*wrappedError),
			wantErr: io.EOF,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.err.Unwrap(); err != tt.wantErr {
				t.Errorf("Unwrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
