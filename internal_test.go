package gobrake

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("findGitDir", func() {
	It("returns first directory containing .git", func() {
		workDir, _ := os.Getwd()
		tests := []struct {
			dir string
			ok  bool
		}{
			{"", true},
			{"./", true},
			{"...", true},
			{"../gobrake", true},
			{workDir, true},
			{filepath.Join(workDir, "internal"), true},
			{filepath.Join(workDir, "internal", "lrucache"), true},
			{"../", false},
			{"abc", false},
			{filepath.Join(workDir, "abc"), false},
			{filepath.Dir(workDir), false},
		}

		for _, test := range tests {
			dir, ok := findGitDir(test.dir)
			if ok {
				Expect(dir).To(Equal(workDir))
				continue
			}

			Expect(dir).To(Equal(""))
		}
	})
})
