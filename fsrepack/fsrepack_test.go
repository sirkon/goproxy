package fsrepack

import (
	"testing"
)

func Test_gitlabRepacker_Relativer(t *testing.T) {
	type fields struct {
		version     int
		projectPath string
	}
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "trivial-regular-file",
			fields: fields{
				version: 1,
			},
			args: args{
				path: "module-12321321/go.mod",
			},
			want:    "go.mod",
			wantErr: false,
		},
		{
			name: "trivial-regular-file-with-level",
			fields: fields{
				version: 1,
			},
			args: args{
				path: "module-12321321/dir/file.go",
			},
			want:    "dir/file.go",
			wantErr: false,
		},
		{
			name: "trivial-directory",
			fields: fields{
				version: 1,
			},
			args: args{
				path: "module-12321321/dir/",
			},
			want:    "dir/",
			wantErr: false,
		},
		{
			name: "trivial-directory-with-level",
			fields: fields{
				version: 1,
			},
			args: args{
				path: "module-12321321/dir1/dir2/",
			},
			want:    "dir1/dir2/",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gitlabRepacker{
				version:     tt.fields.version,
				projectPath: tt.fields.projectPath,
			}
			got, err := r.Relativer(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("gitlabRepacker.Relativer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("gitlabRepacker.Relativer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_gitlabRepacker_Destinator(t *testing.T) {
	type fields struct {
		version     int
		projectPath string
	}
	type args struct {
		path string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "file-trivial",
			fields: fields{
				version:     1,
				projectPath: "gitlab.com/user/module",
			},
			args: args{
				path: "go.mod",
			},
			want: "gitlab.com/user/module/go.mod",
		},
		{
			name: "file-versioned",
			fields: fields{
				version:     21,
				projectPath: "gitlab.com/user/module",
			},
			args: args{
				path: "go.mod",
			},
			want: "gitlab.com/user/module/v21/go.mod",
		},
		{
			name: "file-level-versioned",
			fields: fields{
				version:     12,
				projectPath: "gitlab.com/user/module",
			},
			args: args{
				path: "level/file.go",
			},
			want: "gitlab.com/user/module/v12/level/file.go",
		},
		{
			name: "dir-trivial",
			fields: fields{
				version:     1,
				projectPath: "gitlab.com/user/module",
			},
			args: args{
				path: "dir/",
			},
			want: "gitlab.com/user/module/dir/",
		},
		{
			name: "dir-level-versioned",
			fields: fields{
				version:     12,
				projectPath: "gitlab.com/user/module",
			},
			args: args{
				path: "level/dir/",
			},
			want: "gitlab.com/user/module/v12/level/dir/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gitlabRepacker{
				version:     tt.fields.version,
				projectPath: tt.fields.projectPath,
			}
			if got := r.Destinator(tt.args.path); got != tt.want {
				t.Errorf("gitlabRepacker.Destinator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_standard_Relativer(t *testing.T) {
	type fields struct {
		gitlabRepacker gitlabRepacker
		expectedPrefix string
	}
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "trivial",
			fields: fields{
				gitlabRepacker: gitlabRepacker{
					version: 0,
				},
				expectedPrefix: "gitlab.com/user/module/",
			},
			args: args{
				path: "gitlab.com/user/module/dir/file.go",
			},
			want:    "dir/file.go",
			wantErr: false,
		},
		{
			name: "leveled-dir",
			fields: fields{
				gitlabRepacker: gitlabRepacker{
					version: 12,
				},
				expectedPrefix: "gitlab.com/user/module/",
			},
			args: args{
				path: "gitlab.com/user/module/dir1/dir2/",
			},
			want:    "dir1/dir2/",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := standard{
				gitlabRepacker: tt.fields.gitlabRepacker,
				expectedPrefix: tt.fields.expectedPrefix,
			}
			got, err := r.Relativer(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("standard.Relativer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("standard.Relativer() = %v, want %v", got, tt.want)
			}
		})
	}
}
