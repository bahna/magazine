package slugifier

import (
	"testing"
)

func TestMakeSlug(t *testing.T) {
	slgf := NewSlugifier()
	type args struct {
		title string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"A", args{"Обложки METALBIND"}, "oblozhki-metalbind"},
		{"B", args{"Твердые обложки А4 синие упак. 10пар"}, "tverdye-oblozhki-a4-sinie-upak-10par"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := slgf.Slugify(tt.args.title); got != tt.want {
				t.Errorf("MakeSlug() = %v, want %v", got, tt.want)
			}
		})
	}
}
