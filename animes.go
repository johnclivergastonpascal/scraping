package anime

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Episodio struct {
	Titulo   string `json:"titulo"`
	Numero   string `json:"numero"`
	Fecha    string `json:"fecha"`
	Imagen   string `json:"imagen"`
	URL      string `json:"url"`
	Sinopsis string `json:"sinopsis"`
	VideoURL string `json:"video_url"`
}

type Temporada struct {
	Temporada int        `json:"temporada"`
	Episodios []Episodio `json:"episodios"`
}

type animes struct {
	Year        string      `json:"year"`
	Titulo      string      `json:"titulo"`
	Genero      string      `json:"genero"`
	Imagen      string      `json:"imagen"`
	URL         string      `json:"url"`
	Puntuacion  string      `json:"puntuacion"`
	Wallpaper   string      `json:"wallpaper"`
	Sinopsis    string      `json:"sinopsis"`
	Descripcion string      `json:"descripcion"`
	Temporadas  []Temporada `json:"temporadas"`
}

var userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"

// Función para obtener documento goquery
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

// Obtener total de páginas
func getTotalPages() int {
	doc, err := getDocument("https://sololatino.net/animes/")
	if err != nil {
		log.Printf("❌ Error al obtener la primera página: %v\n", err)
		return 1
	}

	lastPage := 105
	doc.Find("div.pagination a.page-numbers").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if n, err := strconv.Atoi(text); err == nil && n > lastPage {
			lastPage = n
		}
	})
	fmt.Printf("🔢 Total de páginas detectadas: %d\n", lastPage)
	return lastPage
}

// Parsear un episodio (sinopsis y video)
func parseEpisodio(url string) (string, string) {
	doc, err := getDocument(url)
	if err != nil {
		return "No disponible", "No disponible"
	}

	sinopsis := "No disponible"
	doc.Find("div[itemprop='description'].wp-content p").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			sinopsis = s.Text()
		}
	})

	video := "No disponible"
	iframe := doc.Find("iframe.metaframe.rptss")
	src, exists := iframe.Attr("src")
	if exists {
		video = src
	}

	return sinopsis, video
}

// Parsear una serie completa
func parseanimes(url, year, titulo, genero, imagen string) (animes, error) {
	doc, err := getDocument(url)
	if err != nil {
		return animes{}, err
	}

	wallpaper := "No disponible"
	doc.Find("div.wallpaper").Each(func(i int, s *goquery.Selection) {
		style, _ := s.Attr("style")
		if strings.Contains(style, "url(") {
			wallpaper = strings.Split(strings.Split(style, "url(")[1], ")")[0]
		}
	})

	puntuacion := doc.Find("div.nota span").Text()

	sinopsis := "No disponible"
	descripcion := "No disponible"
	doc.Find("div[itemprop='description'].wp-content").Each(func(i int, s *goquery.Selection) {
		s.Find("p").Each(func(j int, p *goquery.Selection) {
			if j == 0 {
				descripcion = p.Text()
			}
		})
		s.Find("h3").Each(func(i int, h *goquery.Selection) {
			sinopsis = h.Text()
		})
	})

	var temporadas []Temporada
	doc.Find("div.se-c[data-season]").Each(func(i int, s *goquery.Selection) {
		_ = s.AttrOr("data-season", "0")
		var episodios []Episodio

		s.Find("ul.episodios li").Each(func(i int, li *goquery.Selection) {
			a := li.Find("a")
			href, _ := a.Attr("href")
			img := a.Find("img").AttrOr("src", "No disponible")
			epst := a.Find("div.epst").Text()
			numerando := a.Find("div.numerando").Text()
			fecha := a.Find("span.date").Text()

			sinopsisEp, videoEp := parseEpisodio(href)

			episodios = append(episodios, Episodio{
				Titulo:   strings.TrimSpace(epst),
				Numero:   strings.TrimSpace(numerando),
				Fecha:    strings.TrimSpace(fecha),
				Imagen:   img,
				URL:      href,
				Sinopsis: sinopsisEp,
				VideoURL: videoEp,
			})
		})

		temporadas = append(temporadas, Temporada{
			Temporada: i + 1,
			Episodios: episodios,
		})
	})

	return animes{
		Year:        year,
		Titulo:      titulo,
		Genero:      genero,
		Imagen:      imagen,
		URL:         url,
		Puntuacion:  puntuacion,
		Wallpaper:   wallpaper,
		Sinopsis:    sinopsis,
		Descripcion: descripcion,
		Temporadas:  temporadas,
	}, nil
}

// Mezclar temporadas nuevas con las existentes
func mergeTemporadas(existing, scraped []Temporada) []Temporada {
	existingMap := make(map[int]int) // temporada num -> índice en existing

	for i, temp := range existing {
		existingMap[temp.Temporada] = i
	}

	for _, sTemp := range scraped {
		if idx, ok := existingMap[sTemp.Temporada]; ok {
			// Actualizar episodios de temporada existente
			existing[idx].Episodios = mergeEpisodios(existing[idx].Episodios, sTemp.Episodios)
		} else {
			// Temporada nueva, agregar
			existing = append(existing, sTemp)
		}
	}

	return existing
}

// Mezclar episodios nuevos con los existentes (sin duplicados)
func mergeEpisodios(existing, scraped []Episodio) []Episodio {
	existingMap := make(map[string]bool) // url de episodio para evitar duplicados

	for _, e := range existing {
		existingMap[e.URL] = true
	}

	for _, s := range scraped {
		if !existingMap[s.URL] {
			existing = append(existing, s)
		}
	}
	return existing
}

func Scrape() {
	var animess []animes
	animesMap := make(map[string]bool)
	jsonFile := "animes.json"

	// Cargar JSON si existe
	if _, err := os.Stat(jsonFile); err == nil {
		file, err := os.Open(jsonFile)
		if err == nil {
			defer file.Close()
			err = json.NewDecoder(file).Decode(&animess)
			if err == nil {
				for _, s := range animess {
					animesMap[s.URL] = true
				}
				fmt.Printf("📁 Cargadas %d animes desde el archivo existente.\n", len(animess))
			}
		}
	}

	totalPages := getTotalPages()

	for page := 1; page <= totalPages; page++ {
		pageURL := fmt.Sprintf("https://sololatino.net/animes/page/%d/", page)
		fmt.Printf("📄 Procesando página %d...\n", page)

		doc, err := getDocument(pageURL)
		if err != nil {
			log.Printf("❌ Error cargando página %d: %v\n", page, err)
			continue
		}

		doc.Find("article").Each(func(i int, s *goquery.Selection) {
			year := strings.TrimSpace(s.Find("p").Text())
			titulo := strings.TrimSpace(s.Find("h3").Text())
			genero := strings.TrimSpace(s.Find("span").Text())
			img, _ := s.Find("img").Attr("data-srcset")
			link, _ := s.Find("a").Attr("href")

			if link != "" {
				serieNueva, err := parseanimes(link, year, titulo, genero, img)
				if err != nil {
					log.Printf("⚠️ Error al procesar %s: %v\n", link, err)
					return
				}

				if animesMap[link] {
					// Actualizar la serie existente
					for i, s := range animess {
						if s.URL == link {
							animess[i].Temporadas = mergeTemporadas(animess[i].Temporadas, serieNueva.Temporadas)
							animess[i].Puntuacion = serieNueva.Puntuacion
							animess[i].Sinopsis = serieNueva.Sinopsis
							fmt.Printf("🔄 Actualizada: %s - ⭐ %s\n", titulo, serieNueva.Puntuacion)
							break
						}
					}
				} else {
					// Agregar nueva serie
					animess = append(animess, serieNueva)
					animesMap[link] = true
					fmt.Printf("🎬 Agregada: %s - ⭐ %s\n", titulo, serieNueva.Puntuacion)
				}
			}
		})
	}

	// Guardar JSON actualizado
	file, err := os.Create(jsonFile)
	if err != nil {
		log.Fatalf("Error creando el archivo JSON: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(animess)
	if err != nil {
		log.Fatalf("Error al escribir el JSON: %v", err)
	}

	fmt.Println("✅ Datos guardados en", jsonFile)
}
