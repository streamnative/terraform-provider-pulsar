package hashcode

import "testing"

func TestString(t *testing.T) {
	type args struct {
		input string
		want  int
	}
	cases := []args{
		{
			input: "",
			want:  0,
		},
	}
	for _, testCase := range cases {
		if got := String(testCase.input); got != testCase.want {
			t.Errorf("String() = %v, want %v", got, testCase.want)
		}
	}
}

func TestStrings(t *testing.T) {
	type args struct {
		strings []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Strings(tt.args.strings); got != tt.want {
				t.Errorf("Strings() = %v, want %v", got, tt.want)
			}
		})
	}
}
