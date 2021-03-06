package cargo_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dmikusa/rust-cargo-cnb/cargo"
	"github.com/dmikusa/rust-cargo-cnb/cargo/mocks"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/scribe"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		layersDir  string
		cnbPath    string
		timestamp  string
		buffer     *bytes.Buffer
		mockRunner mocks.Runner
		clock      chronos.Clock

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		layersDir, err = ioutil.TempDir("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbPath, err = ioutil.TempDir("", "cnb-path")
		Expect(err).NotTo(HaveOccurred())

		now := time.Now()
		clock = chronos.NewClock(func() time.Time { return now })
		timestamp = now.Format(time.RFC3339Nano)
		buffer = bytes.NewBuffer(nil)

		mockRunner = mocks.Runner{}

		logger := scribe.NewEmitter(buffer)

		build = cargo.Build(&mockRunner, clock, logger)
	})

	it.After(func() {
		mockRunner.AssertExpectations(t)

		Expect(os.RemoveAll(workingDir)).To(Succeed())
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbPath)).To(Succeed())
	})

	context("build cases", func() {
		it("builds a single member", func() {
			member, err := url.Parse("file:///workspace")
			Expect(err).ToNot(HaveOccurred())
			mockRunner.On(
				"WorkspaceMembers",
				workingDir,
				mock.AnythingOfType("packit.Layer"),
				mock.AnythingOfType("packit.Layer")).Return([]url.URL{*member}, nil)

			mockRunner.On(
				"Install",
				workingDir,
				mock.AnythingOfType("packit.Layer"),
				mock.AnythingOfType("packit.Layer")).Return(nil)

			Expect(os.MkdirAll(filepath.Join(layersDir, "rust-cargo"), 0755)).ToNot(HaveOccurred())
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "rust"},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(packit.BuildResult{
				Layers: []packit.Layer{
					{
						Name:             "rust-cargo",
						Path:             filepath.Join(layersDir, "rust-cargo"),
						Build:            false,
						Cache:            true,
						Launch:           false,
						SharedEnv:        packit.Environment{},
						BuildEnv:         packit.Environment{},
						LaunchEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Metadata: map[string]interface{}{
							"built_at": timestamp,
						},
					},
					{
						Name:             "rust-bin",
						Path:             filepath.Join(layersDir, "rust-bin"),
						Build:            false,
						Launch:           true,
						Cache:            false,
						SharedEnv:        packit.Environment{},
						BuildEnv:         packit.Environment{},
						LaunchEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Metadata: map[string]interface{}{
							"built_at": timestamp,
						},
					},
				},
			}))
		})

		it("builds a multi-member project", func() {
			member1, err := url.Parse("file:///workspace1")
			Expect(err).ToNot(HaveOccurred())
			member2, err := url.Parse("file:///workspace2")
			Expect(err).ToNot(HaveOccurred())

			mockRunner.On(
				"WorkspaceMembers",
				workingDir,
				mock.AnythingOfType("packit.Layer"),
				mock.AnythingOfType("packit.Layer")).Return([]url.URL{*member1, *member2}, nil)

			mockRunner.On(
				"InstallMember",
				member1.Path,
				workingDir,
				mock.AnythingOfType("packit.Layer"),
				mock.AnythingOfType("packit.Layer")).Return(nil)

			mockRunner.On(
				"InstallMember",
				member2.Path,
				workingDir,
				mock.AnythingOfType("packit.Layer"),
				mock.AnythingOfType("packit.Layer")).Return(nil)

			Expect(os.MkdirAll(filepath.Join(layersDir, "rust-cargo"), 0755)).ToNot(HaveOccurred())
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "rust"},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(packit.BuildResult{
				Layers: []packit.Layer{
					{
						Name:             "rust-cargo",
						Path:             filepath.Join(layersDir, "rust-cargo"),
						Build:            false,
						Cache:            true,
						Launch:           false,
						SharedEnv:        packit.Environment{},
						BuildEnv:         packit.Environment{},
						LaunchEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Metadata: map[string]interface{}{
							"built_at": timestamp,
						},
					},
					{
						Name:             "rust-bin",
						Path:             filepath.Join(layersDir, "rust-bin"),
						Build:            false,
						Launch:           true,
						Cache:            false,
						SharedEnv:        packit.Environment{},
						BuildEnv:         packit.Environment{},
						LaunchEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Metadata: map[string]interface{}{
							"built_at": timestamp,
						},
					},
				},
			}))
		})

		it("builds a multi-member project with single member after filter", func() {
			member1, err := url.Parse("file:///workspace1")
			Expect(err).ToNot(HaveOccurred())

			// this filters down to one member
			mockRunner.On(
				"WorkspaceMembers",
				workingDir,
				mock.AnythingOfType("packit.Layer"),
				mock.AnythingOfType("packit.Layer")).Return([]url.URL{*member1}, nil)

			mockRunner.On(
				"InstallMember",
				member1.Path,
				workingDir,
				mock.AnythingOfType("packit.Layer"),
				mock.AnythingOfType("packit.Layer")).Return(nil)

			Expect(os.MkdirAll(filepath.Join(layersDir, "rust-cargo"), 0755)).ToNot(HaveOccurred())
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "rust"},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(packit.BuildResult{
				Layers: []packit.Layer{
					{
						Name:             "rust-cargo",
						Path:             filepath.Join(layersDir, "rust-cargo"),
						Build:            false,
						Cache:            true,
						Launch:           false,
						SharedEnv:        packit.Environment{},
						BuildEnv:         packit.Environment{},
						LaunchEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Metadata: map[string]interface{}{
							"built_at": timestamp,
						},
					},
					{
						Name:             "rust-bin",
						Path:             filepath.Join(layersDir, "rust-bin"),
						Build:            false,
						Launch:           true,
						Cache:            false,
						SharedEnv:        packit.Environment{},
						BuildEnv:         packit.Environment{},
						LaunchEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Metadata: map[string]interface{}{
							"built_at": timestamp,
						},
					},
				},
			}))
		})
	})

	context("failure cases", func() {
		context("when the rust layer cannot be retrieved", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(filepath.Join(layersDir, "rust-cargo.toml"), nil, 0000)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "rust"},
						},
					},
				})
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("cargo build fails", func() {
			it.Before(func() {
				mockRunner := mocks.Runner{}
				mockRunner.On(
					"Install",
					workingDir,
					mock.AnythingOfType("packit.Layer"),
					mock.AnythingOfType("packit.Layer"),
				).Return(fmt.Errorf("expected"))

				member, err := url.Parse("file:///workspace")
				Expect(err).ToNot(HaveOccurred())
				mockRunner.On(
					"WorkspaceMembers",
					workingDir,
					mock.AnythingOfType("packit.Layer"),
					mock.AnythingOfType("packit.Layer")).Return([]url.URL{*member}, nil)

				logger := scribe.NewEmitter(buffer)

				build = cargo.Build(&mockRunner, clock, logger)
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbPath,
					Stack:      "some-stack",
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "rust"},
						},
					},
				})
				Expect(err).To(MatchError("expected"))
			})
		})

		context("cargo cannot fetch members", func() {
			it.Before(func() {
				mockRunner := mocks.Runner{}

				mockRunner.On(
					"WorkspaceMembers",
					workingDir,
					mock.AnythingOfType("packit.Layer"),
					mock.AnythingOfType("packit.Layer")).Return(nil, fmt.Errorf("broken"))

				logger := scribe.NewEmitter(buffer)

				build = cargo.Build(&mockRunner, clock, logger)
			})

			it.After(func() {
				mockRunner.AssertExpectations(t)
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbPath,
					Stack:      "some-stack",
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "rust"},
						},
					},
				})
				Expect(err).To(MatchError("broken"))
			})
		})
	})
}
