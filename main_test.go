package main

import (
	"testing"
)

func Test_increaseCode(t *testing.T) {
	type args struct {
		code string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "increaseCode",
			args: args{
				code: "ab1",
			},
			want:    "ab2",
			wantErr: false,
		},
		{
			name: "increaseCode",
			args: args{
				code: "ab9",
			},
			want:    "aca",
			wantErr: false,
		},
		{
			name: "increaseCode",
			args: args{
				code: "az9",
			},
			want:    "a0a",
			wantErr: false,
		},
		{
			name: "increaseCode",
			args: args{
				code: "zz9",
			},
			want:    "z0a",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := increaseCode(tt.args.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("increaseCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("increaseCode() got = %v, want %v", got, tt.want)
			}
		})
	}
}
