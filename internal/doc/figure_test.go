package doc

import "testing"

func figureOpts() Options {
	return Options{Images: true, CellW: 8, CellH: 16, ScreenRows: 48, BaseDir: "../../testdata"}
}

func TestFiguresKeepTheirShape(t *testing.T) {
	d := Render([]byte("![wide](images/landscape.png)\n"), 100, figureOpts())
	if len(d.Figures) != 1 {
		t.Fatalf("got %d figures, want 1", len(d.Figures))
	}
	f := d.Figures[0]
	// 1600×900 at 8×16 px cells: rows ≈ cols × 8 × 900 / 1600 / 16.
	want := float64(f.Cols) * 8 * 900 / 1600 / 16
	if diff := float64(f.Rows) - want; diff > 1 || diff < -1 {
		t.Errorf("figure %d×%d cells, want about %.1f rows for its shape", f.Cols, f.Rows, want)
	}
}

func TestTallFiguresAreCapped(t *testing.T) {
	d := Render([]byte("![tall](images/tree.png)\n"), 100, figureOpts())
	f := d.Figures[0]
	if f.Rows > 48/3 {
		t.Errorf("tall figure is %d rows, want at most a third of 48", f.Rows)
	}
	want := float64(f.Rows) * 16 * 600 / 1200 / 8
	if diff := float64(f.Cols) - want; diff > 1 || diff < -1 {
		t.Errorf("capped figure %d×%d cells lost its shape (want about %.1f cols)", f.Cols, f.Rows, want)
	}
}

func TestSmallImagesAreNotScaledUp(t *testing.T) {
	d := Render([]byte("![icon](images/icon.png)\n"), 100, figureOpts())
	if f := d.Figures[0]; f.Cols > 6 {
		t.Errorf("48 px icon is %d cells wide, want at most 6 (48/8)", f.Cols)
	}
}

func TestInlineImagesAreChips(t *testing.T) {
	d := Render([]byte("Text ![badge](images/icon.png) more.\n"), 100, figureOpts())
	if len(d.Figures) != 0 {
		t.Errorf("an image inside a sentence became a figure")
	}
}

func TestMissingAndRemoteImagesAreCards(t *testing.T) {
	d := Render([]byte("![a](missing.png)\n\n![b](https://example.com/x.png)\n"), 100, figureOpts())
	if len(d.Figures) != 0 {
		t.Errorf("missing or remote images became figures")
	}
}
