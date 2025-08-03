package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type ExtraInfo struct {
	Recaudacion   string `json:"r"`
	Pais          string `json:"p"`
	Duracion      string `json:"d"`
	Clasificacion string `json:"c"`
}

type Persona struct {
	Nombre    string `json:"n"`
	URL       string `json:"u"`
	Imagen    string `json:"i"`
	Personaje string `json:"personaje,omitempty"`
}

type Pelicula struct {
	Year        string    `json:"y"`
	Titulo      string    `json:"t"`
	Genero      string    `json:"g"`
	Imagen      string    `json:"i"`
	URL         string    `json:"u"`
	Puntuacion  string    `json:"p"`
	Wallpaper   string    `json:"w"`
	Sinopsis    string    `json:"s"`
	Descripcion string    `json:"d"`
	ExtraInfo   ExtraInfo `json:"e"`
	Director    Persona   `json:"dr"`
	Reparto     []Persona `json:"r"`
	VideoURL    string    `json:"v"`
}

var userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"

func getDocument(url string) (*goquery.Document, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", userAgent)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code: %d", res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func parseDetalle(url string, img string, year string, titulo string, genero string) (Pelicula, error) {
	doc, err := getDocument(url)
	if err != nil {
		return Pelicula{}, err
	}

	// Wallpaper
	wallpaper := "No disponible"
	doc.Find("div.wallpaper").Each(func(i int, s *goquery.Selection) {
		style, _ := s.Attr("style")
		if strings.Contains(style, "url(") {
			wallpaper = strings.Split(strings.Split(style, "url(")[1], ")")[0]
		}
	})

	// Puntuación
	puntuacion := doc.Find("div.nota span").Text()

	// Descripción y sinopsis
	sinopsis := "No disponible"    // Valor por defecto
	descripcion := "No disponible" // Valor por defecto

	// Encuentra el div de la descripción
	descripcionDiv := doc.Find("div[itemprop='description'].wp-content")

	// Si se encuentra el div de descripción, extraer los párrafos
	descripcionDiv.Each(func(i int, s *goquery.Selection) {
		s.Find("p").Each(func(j int, p *goquery.Selection) {
			// Verifica si el párrafo tiene la clase "tagline" para la sinopsis
			if p.HasClass("tagline") && sinopsis == "No disponible" {
				sinopsis = p.Text() // Asigna la sinopsis
			} else if descripcion == "No disponible" {
				descripcion = p.Text() // Asigna la descripción
			}
		})
	})

	// Fallback: si solo hay un párrafo sin tagline, úsalo para ambos
	if descripcionDiv.Length() == 1 && sinopsis == "No disponible" && descripcion == "No disponible" {
		texto := descripcionDiv.Text()
		sinopsis = texto
		descripcion = texto
	}

	// Extra info
	extra := ExtraInfo{
		Recaudacion:   "No disponible",
		Pais:          "No disponible",
		Duracion:      "No disponible",
		Clasificacion: "No disponible",
	}
	extraBox := doc.Find("div.sbox.extra")
	extra.Recaudacion = extraBox.Find("span.tagline span").Text()
	extra.Pais = extraBox.Find("span.country").Text()
	extra.Duracion = extraBox.Find("span.runtime").Text()
	extra.Clasificacion = extraBox.Find("span.rated").Text()

	// Director
	director := Persona{}
	directorDiv := doc.Find("div.sbox.srepart div[itemprop='director']")
	if directorDiv.Length() > 0 {
		director.Nombre, _ = directorDiv.Find("meta[itemprop='name']").Attr("content")
		director.URL, _ = directorDiv.Find("a").Attr("href")
		director.Imagen, _ = directorDiv.Find("img").Attr("src")
	}

	// Reparto
	var reparto []Persona
	doc.Find("div#actores_div div[itemprop='actor']").Each(func(i int, s *goquery.Selection) {
		nombre, _ := s.Find("meta[itemprop='name']").Attr("content")
		url, _ := s.Find("a").Attr("href")
		imagen, _ := s.Find("img").Attr("src")
		personaje := s.Find("div.caracter").Text()
		reparto = append(reparto, Persona{
			Nombre:    nombre,
			URL:       url,
			Imagen:    imagen,
			Personaje: personaje,
		})
	})

	// Video URL
	videoURL := "No disponible"
	iframe := doc.Find("iframe.metaframe.rptss")
	src, exists := iframe.Attr("src")
	if exists {
		videoURL = src
	}

	return Pelicula{
		Year:        year,
		Titulo:      titulo,
		Genero:      genero,
		Imagen:      img,
		URL:         url,
		Puntuacion:  puntuacion,
		Wallpaper:   wallpaper,
		Sinopsis:    sinopsis,
		Descripcion: descripcion,
		ExtraInfo:   extra,
		Director:    director,
		Reparto:     reparto,
		VideoURL:    videoURL,
	}, nil
}

func Movies() {
	var peliculas []Pelicula
	peliculasMap := make(map[string]bool)

	// Leer JSON existente si existe
	jsonFile := "peliculas.json"
	if _, err := os.Stat(jsonFile); err == nil {
		file, err := os.Open(jsonFile)
		if err == nil {
			defer file.Close()
			err = json.NewDecoder(file).Decode(&peliculas)
			if err == nil {
				for _, p := range peliculas {
					peliculasMap[p.URL] = true
				}
				fmt.Printf("📁 Cargadas %d películas desde el archivo existente.\n", len(peliculas))
			}
		}
	}

	page := 1
	for {
		pageURL := fmt.Sprintf("https://sololatino.net/peliculas/page/%d/", page)
		fmt.Printf("📄 Procesando página %d...\n", page)

		doc, err := getDocument(pageURL)
		if err != nil {
			if strings.Contains(err.Error(), "status code: 404") {
				fmt.Println("🚫 Página no encontrada. Fin del scraping.")
				break
			}
			log.Printf("❌ Error cargando la página: %v\n", err)
			page++
			continue
		}

		doc.Find("article").Each(func(i int, s *goquery.Selection) {
			year := strings.TrimSpace(s.Find("p").Text())
			titulo := strings.TrimSpace(s.Find("h3").Text())
			genero := strings.TrimSpace(s.Find("span").Text())
			img, _ := s.Find("img").Attr("data-srcset")
			link, _ := s.Find("a").Attr("href")

			if link != "" && !peliculasMap[link] {
				pelicula, err := parseDetalle(link, img, year, titulo, genero)
				if err == nil {
					peliculas = append(peliculas, pelicula)
					peliculasMap[link] = true
					fmt.Printf("🎬 Agregada: %s - ⭐ %s\n", titulo, pelicula.Puntuacion)
				} else {
					log.Printf("⚠️ Error al procesar %s: %v\n", link, err)
				}
			} else if peliculasMap[link] {
				fmt.Printf("⏩ Saltando duplicado: %s\n", titulo)
			}
		})

		page++
	}

	// Guardar en JSON
	file, err := os.Create(jsonFile)
	if err != nil {
		log.Fatalf("Error creando el archivo JSON: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(peliculas)
	if err != nil {
		log.Fatalf("Error al escribir el JSON: %v", err)
	}

	fmt.Println("✅ Datos actualizados en", jsonFile)
}
