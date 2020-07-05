package autoscan

import "testing"

func TestAddPrefix(t *testing.T) {
	type Test struct {
		Name     string
		Prefix   string
		Path     string
		Expected string
	}

	var testCases = []Test{
		{
			Name:     "Normal case",
			Prefix:   "/mnt/unionfs",
			Path:     "/Media/Movies",
			Expected: "/mnt/unionfs/Media/Movies",
		},
		{
			Name:     "Slash at the end of the path is dropped",
			Prefix:   "/mnt/",
			Path:     "media/movies/",
			Expected: "/mnt/media/movies",
		},
		{
			Name:     "Empty prefix does not affect path",
			Prefix:   "",
			Path:     "/media/movies",
			Expected: "/media/movies",
		},
		{
			Name:     "Prefix without slash is rewritten to include the slash",
			Prefix:   "mnt/unionfs",
			Path:     "/media/movies",
			Expected: "/mnt/unionfs/media/movies",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Expected, func(t *testing.T) {
			result := AddPrefix(tc.Prefix, tc.Path)
			if result != tc.Expected {
				t.Errorf("%s does not equal %s", result, tc.Expected)
			}
		})
	}
}

func TestStripPrefix(t *testing.T) {
	type Test struct {
		Name     string
		Prefix   string
		Path     string
		Expected string
	}

	var testCases = []Test{
		{
			Name:     "Normal case",
			Prefix:   "/mnt/unionfs",
			Path:     "/mnt/unionfs/Movies/movie",
			Expected: "/Movies/movie",
		},
		{
			Name:     "Slash is added in front",
			Prefix:   "/hello/",
			Path:     "/hello/world",
			Expected: "/world",
		},
		{
			Name:     "Empty prefix does not affect path",
			Prefix:   "",
			Path:     "/hello/world",
			Expected: "/hello/world",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Expected, func(t *testing.T) {
			result := StripPrefix(tc.Prefix, tc.Path)
			if result != tc.Expected {
				t.Errorf("%s does not equal %s", result, tc.Expected)
			}
		})
	}
}
