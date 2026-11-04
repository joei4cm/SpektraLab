package dev.filmprint.lab

import android.graphics.Bitmap
import android.net.Uri
import android.os.Bundle
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.compose.setContent
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme {
                AppScreen()
            }
        }
    }
}

private fun buildParams(film: String, paper: String): String =
    """{"film_stock":"$film","print_stock":"$paper",""" +
        """"input_color_space":"sRGB","input_cctf_decoding":true,""" +
        """"output_color_space":"sRGB","output_cctf_encoding":true}"""

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun AppScreen() {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()

    var engineStatus by remember { mutableStateOf("Preparing engine...") }
    var engineReady by remember { mutableStateOf(false) }
    var source by remember { mutableStateOf<SourceImage?>(null) }
    var rendered by remember { mutableStateOf<Bitmap?>(null) }
    var film by remember { mutableStateOf(Profiles.defaultFilm) }
    var paper by remember { mutableStateOf(Profiles.defaultPaper) }
    var busy by remember { mutableStateOf(false) }
    var statusLine by remember { mutableStateOf("") }

    androidx.compose.runtime.LaunchedEffect(Unit) {
        withContext(Dispatchers.Default) {
            try {
                val dir = EngineAssets.ensureExtracted(context)
                val err = FilmEngine.create(dir.absolutePath)
                if (err == null) {
                    engineReady = true
                    engineStatus = "Engine ready · ${FilmEngine.buildInfo()}"
                } else {
                    engineStatus = "Engine failed: $err"
                }
            } catch (t: Throwable) {
                engineStatus = "Engine error: ${t.message}"
            }
        }
    }

    val picker = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri: Uri? ->
        if (uri != null) {
            scope.launch {
                busy = true
                statusLine = "Decoding image..."
                val img = withContext(Dispatchers.Default) { ImageIo.decode(context, uri) }
                busy = false
                if (img == null) {
                    statusLine = "Could not decode that image."
                } else {
                    source = img
                    rendered = null
                    statusLine = "Loaded ${img.width}×${img.height}. Choose film + paper, then Render."
                }
            }
        }
    }

    fun runRender() {
        val img = source ?: return
        scope.launch {
            busy = true
            statusLine = "Rendering ${film.label} → ${paper.label}..."
            val result = withContext(Dispatchers.Default) {
                val floats = ImageIo.toLinearFloats(img.bitmap)
                FilmEngine.render(floats, img.width, img.height, buildParams(film.id, paper.id), "full")
            }
            busy = false
            if (result.error != null || result.argb == null) {
                statusLine = "Render failed: ${result.error ?: "no pixels"}"
            } else {
                rendered = ImageIo.bitmapFromArgb(result.argb, result.width, result.height)
                statusLine = "Rendered ${result.width}×${result.height}."
            }
        }
    }

    fun exportResult() {
        val bmp = rendered ?: return
        scope.launch {
            val name = "filmprint_${film.id}_${paper.id}_${System.currentTimeMillis()}.png"
            val uri = withContext(Dispatchers.Default) { ImageIo.saveToGallery(context, bmp, name) }
            statusLine = if (uri != null) "Saved to Pictures/FilmPrint Lab" else "Export failed."
            Toast.makeText(context, statusLine, Toast.LENGTH_SHORT).show()
        }
    }

    Scaffold(topBar = { TopAppBar(title = { Text("FilmPrint Lab") }) }) { inner ->
        Column(
            modifier = Modifier
                .padding(inner)
                .padding(16.dp)
                .fillMaxSize()
                .verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(engineStatus, style = MaterialTheme.typography.bodySmall, fontFamily = FontFamily.Monospace)

            Button(onClick = { picker.launch("image/*") }, modifier = Modifier.fillMaxWidth()) {
                Text("Pick image (JPEG / PNG)")
            }

            StockSelector("Film", Profiles.films, film) { film = it }
            StockSelector("Paper", Profiles.papers, paper) { paper = it }

            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Button(
                    onClick = { runRender() },
                    enabled = engineReady && source != null && !busy,
                    modifier = Modifier.fillMaxWidth(0.5f),
                ) { Text("Render") }
                OutlinedButton(
                    onClick = { exportResult() },
                    enabled = rendered != null && !busy,
                    modifier = Modifier.fillMaxWidth(),
                ) { Text("Export") }
            }

            if (statusLine.isNotBlank()) {
                Text(statusLine, style = MaterialTheme.typography.bodyMedium)
            }

            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(360.dp),
                contentAlignment = Alignment.Center,
            ) {
                when {
                    busy -> CircularProgressIndicator()
                    rendered != null -> Image(
                        bitmap = rendered!!.asImageBitmap(),
                        contentDescription = "Rendered print",
                        modifier = Modifier.fillMaxWidth(),
                    )
                    source != null -> Image(
                        bitmap = source!!.bitmap.asImageBitmap(),
                        contentDescription = "Source image",
                        modifier = Modifier.fillMaxWidth(),
                    )
                    else -> Text("No image yet.", style = MaterialTheme.typography.bodyMedium)
                }
            }
            Spacer(Modifier.height(8.dp))
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun StockSelector(
    label: String,
    options: List<Stock>,
    selected: Stock,
    onSelected: (Stock) -> Unit,
) {
    var expanded by remember { mutableStateOf(false) }
    ExposedDropdownMenuBox(
        expanded = expanded,
        onExpandedChange = { expanded = it },
        modifier = Modifier.fillMaxWidth(),
    ) {
        OutlinedTextField(
            value = selected.label,
            onValueChange = {},
            readOnly = true,
            label = { Text(label) },
            trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = expanded) },
            modifier = Modifier
                .menuAnchor()
                .fillMaxWidth(),
        )
        ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            options.forEach { option ->
                DropdownMenuItem(
                    text = { Text(option.label + if (option.monochrome) "  (B&W)" else "") },
                    onClick = {
                        onSelected(option)
                        expanded = false
                    },
                )
            }
        }
    }
}
