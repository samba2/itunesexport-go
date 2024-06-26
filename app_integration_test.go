package main

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const FileContent string = "42"

func TestExportPlaylists(t *testing.T) {
	// arrange
	outputDir := createTempDir(t, "itunes-exporter-test")
	defer os.RemoveAll(outputDir)

	musicFile, musicFileName := prepareMusicFile(t)
	defer os.Remove(musicFile)

	// make sure we have a path only containing "/" as separators
	musicFilePath := filepath.ToSlash(musicFile)
	itunesDbFile := prepareItunesDbFile(t, musicFilePath)
	defer os.Remove(itunesDbFile)

	// act

	// Save the real os.Args and defer the restoration.
	realArgs := os.Args
	defer func() { os.Args = realArgs }()

	// Set the necessary parameters to simulate command line arguments.
	os.Args = []string{
		"itunesexport", // The program name (os.Args[0]).
		"-library", itunesDbFile,
		"-output", outputDir,
		"-type", "M3U",
		"-includeAll",
		"-copy", "PLAYLIST",
	}
	main()

	// assert
	assertPlaylistExportedSuccessfully(t, outputDir, musicFileName)
}

func TestExportPlaylistsWithAdjustedMusicPath(t *testing.T) {
	// arrange
	outputDir := createTempDir(t, "itunes-exporter-test")
	defer os.RemoveAll(outputDir)

	musicFile, musicFileName := prepareMusicFile(t)
	defer os.Remove(musicFile)

	musicFileDir := filepath.Dir(musicFile)
	invalidMusicFilePath := filepath.ToSlash(filepath.Join("/invalid", "path", musicFileName))

	itunesDbFile := prepareItunesDbFile(t, invalidMusicFilePath)
	defer os.Remove(itunesDbFile)

	// act

	// Save the real os.Args and defer the restoration.
	realArgs := os.Args
	defer func() { os.Args = realArgs }()

	// Set the necessary parameters to simulate command line arguments.
	os.Args = []string{
		"itunesexport", // The program name (os.Args[0]).
		"-library", itunesDbFile,
		"-output", outputDir,
		"-type", "M3U",
		"-includeAll",
		"-copy", "PLAYLIST",
		"-musicPath", musicFileDir,   // new music path should be the old/ correct one
		"-musicPathOrig", "/invalid/path",
	}
	main()

	// assert
	assertPlaylistExportedSuccessfully(t, outputDir, musicFileName)
}

func assertPathExists(t *testing.T, path string) {
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("File '%s' does not exist: %v", path, err)
	}
}

func createTempFile(t *testing.T, pattern string) string {
	tmpFile, err := os.CreateTemp("", pattern)
	if err != nil {
		t.Fatal(err)
	}
	return tmpFile.Name()
}

func createTempDir(t *testing.T, pattern string) string {
	tmpDirPath, err := os.MkdirTemp("", pattern)
	if err != nil {
		t.Fatal(err)
	}
	return tmpDirPath
}

func writeFile(t *testing.T, filePath string, content string) {
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write '%s' to file '%s': %v", content, filePath, err)
	}
}

func readFile(t *testing.T, filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file '%s': %v", filePath, err)
	}
	return string(content)
}

func prepareMusicFile(t *testing.T) (string, string) {
	musicFile := createTempFile(t, "Some_Song_*.mp3")
	writeFile(t, musicFile, FileContent)
	return musicFile, filepath.Base(musicFile)
}

func prepareItunesDbFile(t *testing.T, musicFilePath string) string {
	itunesDbContent := readFile(t, "fixture/example-itunes-db.xml")
	itunesDbContentAdjusted := strings.ReplaceAll(string(itunesDbContent), "REPLACE_ME_EXAMPLE_SONG_LOCATION", "file://"+musicFilePath)

	itunesDbFile := createTempFile(t, "testItunesDb_*.xml")
	writeFile(t, itunesDbFile, itunesDbContentAdjusted)

	return itunesDbFile
}

func assertPlaylistExportedSuccessfully(t *testing.T, outputDir string, musicFileName string) {
	expectedPlaylistDir := filepath.Join(outputDir, "My Playlist")
	assertPathExists(t, expectedPlaylistDir)

	expectedCopiedMusicFilePath := filepath.Join(expectedPlaylistDir, musicFileName)
	assertPathExists(t, expectedCopiedMusicFilePath)

	musicFileContent := readFile(t, expectedCopiedMusicFilePath)
	if musicFileContent != FileContent {
		t.Errorf("Content of copied file not as expected. Expected: %s, Got: %s", FileContent, musicFileContent)
	}

	expectedPlaylistFilePath := filepath.Join(outputDir, "My Playlist.m3u")
	assertPlaylistFileCorrectlyWritten(t, expectedPlaylistFilePath, expectedCopiedMusicFilePath)
}

func assertPlaylistFileCorrectlyWritten(t *testing.T, playlistPath string, singleLineContent string) {
	assertPathExists(t, playlistPath)

	playlistFileContents := readFile(t, playlistPath)
	re := buildStringOnSingleLineRegex(singleLineContent)
	matches := re.FindAllString(string(playlistFileContents), -1)

	if len(matches) != 1 {
		t.Errorf("Expected playlist to contain '%s' exactly once, but found %d", singleLineContent, len(matches))
	}
}

// e.g. ...\n/path/to/file.mp3\n
func buildStringOnSingleLineRegex(s string) *regexp.Regexp {
	pattern := "\r?\n" + regexp.QuoteMeta(s) + "\r?\n"
	return regexp.MustCompile(pattern)
}
