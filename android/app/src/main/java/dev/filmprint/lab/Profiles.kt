package dev.filmprint.lab

/** A film or paper profile, identified by the engine's profile-JSON basename. */
data class Stock(val id: String, val label: String, val monochrome: Boolean = false)

/**
 * A curated subset of the profiles bundled in assets/engine/profiles. The ids
 * match the JSON basenames the engine loads (engine/resources/profiles and
 * engine/resources_product/profiles). Colour and B&W are kept as sensible
 * film+paper pairs.
 */
object Profiles {
    val films: List<Stock> = listOf(
        Stock("kodak_portra_400", "Kodak Portra 400"),
        Stock("kodak_portra_160", "Kodak Portra 160"),
        Stock("kodak_portra_800", "Kodak Portra 800"),
        Stock("kodak_ektar_100", "Kodak Ektar 100"),
        Stock("kodak_gold_200", "Kodak Gold 200"),
        Stock("kodak_ultramax_400", "Kodak UltraMax 400"),
        Stock("fujifilm_pro_400h", "Fujifilm Pro 400H"),
        Stock("fujifilm_c200", "Fujifilm C200"),
        Stock("cinestill_800t", "CineStill 800T"),
        Stock("kodak_tri_x_400", "Kodak Tri-X 400", monochrome = true),
        Stock("ilford_hp5_plus_400", "Ilford HP5 Plus 400", monochrome = true),
    )

    val papers: List<Stock> = listOf(
        Stock("kodak_portra_endura", "Kodak Portra Endura"),
        Stock("kodak_supra_endura", "Kodak Supra Endura"),
        Stock("kodak_ultra_endura", "Kodak Ultra Endura"),
        Stock("kodak_endura_premier", "Kodak Endura Premier"),
        Stock("ilford_multigrade_iv_rc", "Ilford Multigrade IV RC", monochrome = true),
    )

    val defaultFilm = films.first()
    val defaultPaper = papers.first()
}
