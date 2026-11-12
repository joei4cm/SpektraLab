package main

// Stock is a film or paper profile id (engine profile JSON basename).
type Stock struct {
	ID          string
	Label       string
	Monochrome  bool
	Stage       string // "filming" | "printing"
}

// Curated subset of engine/resources (+ product) profiles. IDs match the
// engine's profile basenames. Colour and B&W stay as sensible pairs.
var films = []Stock{
	{ID: "kodak_portra_400", Label: "Kodak Portra 400", Stage: "filming"},
	{ID: "kodak_portra_160", Label: "Kodak Portra 160", Stage: "filming"},
	{ID: "kodak_portra_800", Label: "Kodak Portra 800", Stage: "filming"},
	{ID: "kodak_ektar_100", Label: "Kodak Ektar 100", Stage: "filming"},
	{ID: "kodak_gold_200", Label: "Kodak Gold 200", Stage: "filming"},
	{ID: "kodak_ultramax_400", Label: "Kodak UltraMax 400", Stage: "filming"},
	{ID: "fujifilm_pro_400h", Label: "Fujifilm Pro 400H", Stage: "filming"},
	{ID: "fujifilm_c200", Label: "Fujifilm C200", Stage: "filming"},
	{ID: "cinestill_800t", Label: "CineStill 800T", Stage: "filming"},
	{ID: "kodak_tri_x_400", Label: "Kodak Tri-X 400", Monochrome: true, Stage: "filming"},
	{ID: "ilford_hp5_plus_400", Label: "Ilford HP5 Plus 400", Monochrome: true, Stage: "filming"},
}

var papers = []Stock{
	{ID: "kodak_portra_endura", Label: "Kodak Portra Endura", Stage: "printing"},
	{ID: "kodak_supra_endura", Label: "Kodak Supra Endura", Stage: "printing"},
	{ID: "kodak_ultra_endura", Label: "Kodak Ultra Endura", Stage: "printing"},
	{ID: "kodak_endura_premier", Label: "Kodak Endura Premier", Stage: "printing"},
	{ID: "ilford_multigrade_iv_rc", Label: "Ilford Multigrade IV RC", Monochrome: true, Stage: "printing"},
}

func defaultFilm() Stock  { return films[0] }
func defaultPaper() Stock { return papers[0] }

func canPrint(film, paper Stock) bool {
	return film.Monochrome == paper.Monochrome
}
