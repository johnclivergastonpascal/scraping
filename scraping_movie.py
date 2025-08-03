import requests
from bs4 import BeautifulSoup
import json

total_pages = 1
peliculas = []

headers = {
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36"
}

for count_page in range(1, total_pages + 1):
    page = f"https://sololatino.net/peliculas/page/{count_page}/"
    print(f"📄 Procesando página {count_page}...")

    try:
        response = requests.get(page, headers=headers)

        if response.status_code == 200:
            soup = BeautifulSoup(response.text, "html.parser")
            articles = soup.find_all("article")

            for article in articles:
                year = article.find("p")
                title = article.find("h3")
                genero = article.find("span")
                img = article.find("img")
                link = article.find("a")

                if all([year, title, genero, img, link]):
                    url_pelicula = link.get("href")

                    try:
                        detalle_res = requests.get(url_pelicula, headers=headers)
                        if detalle_res.status_code == 200:
                            detalle_soup = BeautifulSoup(detalle_res.text, "html.parser")

                            # Fondo de la película
                            wallpaper_div = detalle_soup.find("div", class_="wallpaper")
                            wallpaper_url = "No disponible"
                            if wallpaper_div and "style" in wallpaper_div.attrs:
                                style = wallpaper_div["style"]
                                start = style.find("url(")
                                end = style.find(")", start)
                                if start != -1 and end != -1:
                                    wallpaper_url = style[start + 4:end].strip()

                            # Género
                            genero = genero.text.strip() if genero else "No disponible"

                            # Puntuación
                            puntuacion_div = detalle_soup.find("div", class_="nota")
                            puntuacion = "No disponible"
                            if puntuacion_div and puntuacion_div.find("span"):
                                puntuacion = puntuacion_div.find("span").text.strip()

                            # Descripción y sinopsis
                            descripcion_div = detalle_soup.find("div", itemprop="description", class_="wp-content")
                            sinopsis = "No disponible"
                            description = "No disponible"

                            if descripcion_div:
                                ps = descripcion_div.find_all("p")
                                for p in ps:
                                    classes = p.get("class", [])
                                    texto = p.text.strip()

                                    if not texto:
                                        continue

                                    if "tagline" in classes and sinopsis == "No disponible":
                                        sinopsis = texto
                                    elif description == "No disponible":
                                        description = texto

                                # Fallback: si solo hay un párrafo sin tagline, úsalo para ambos
                                if len(ps) == 1 and sinopsis == "No disponible" and description == "No disponible":
                                    texto = ps[0].text.strip()
                                    sinopsis = texto
                                    description = texto

                            # Información adicional
                            extra_div = detalle_soup.find("div", class_="sbox extra")
                            extra_info = {
                                "recaudacion": "No disponible",
                                "pais": "No disponible",
                                "duracion": "No disponible",
                                "clasificacion": "No disponible"
                            }
                            if extra_div:
                                recaudacion_tag = extra_div.find("span", class_="tagline")
                                if recaudacion_tag and recaudacion_tag.find("span"):
                                    extra_info["recaudacion"] = recaudacion_tag.find("span").text.strip()

                                pais_tag = extra_div.find("span", class_="country")
                                if pais_tag:
                                    extra_info["pais"] = pais_tag.text.strip()

                                duracion_tag = extra_div.find("span", class_="runtime")
                                if duracion_tag:
                                    extra_info["duracion"] = duracion_tag.text.strip()

                                clasificacion_tag = extra_div.find("span", class_="rated")
                                if clasificacion_tag:
                                    extra_info["clasificacion"] = clasificacion_tag.text.strip()

                            # Imágenes
                            imagen_url = img.get("data-srcset") if img else "No disponible"

                            # Director y reparto
                            director = {}
                            reparto = []

                            srepart_div = detalle_soup.find("div", class_="sbox srepart")
                            if srepart_div:
                                # Director
                                director_div = srepart_div.find("div", itemprop="director")
                                if director_div:
                                    nombre = director_div.find("meta", itemprop="name")["content"].strip()
                                    url = director_div.find("a")["href"].strip()
                                    imagen = director_div.find("img")["src"].strip()
                                    director = {
                                        "nombre": nombre,
                                        "url": url,
                                        "imagen": imagen
                                    }

                                # Reparto
                                actores_div = srepart_div.find("div", id="actores_div")
                                if actores_div:
                                    actores = actores_div.find_all("div", itemprop="actor")
                                    for actor in actores:
                                        nombre = actor.find("meta", itemprop="name")["content"].strip()
                                        personaje = actor.find("div", class_="caracter").text.strip()
                                        url = actor.find("a")["href"].strip()
                                        imagen = actor.find("img")["src"].strip()
                                        reparto.append({
                                            "nombre": nombre,
                                            "personaje": personaje,
                                            "url": url,
                                            "imagen": imagen
                                        })

                            # iFrame (Video)
                            iframe = detalle_soup.find("iframe", class_="metaframe rptss")
                            video_url = "No disponible"
                            if iframe and iframe.has_attr('src'):
                                video_url = iframe['src']

                            # Agregar los datos de la película
                            peliculas.append({
                                "year": year.text.strip(),
                                "titulo": title.text.strip(),
                                "genero": genero,
                                "imagen": imagen_url,
                                "url": url_pelicula,
                                "puntuacion": puntuacion,
                                "wallpaper": wallpaper_url,
                                "sinopsis": sinopsis,
                                "descripcion": description,
                                "extra_info": extra_info,
                                "director": director,
                                "reparto": reparto,
                                "video_url": video_url
                            })
                            print(f"🎬 Agregada: {title.text.strip()} - ⭐ {puntuacion}")
                        else:
                            print(f"⚠️ No se pudo acceder al detalle de la película: {url_pelicula}")
                    except Exception as e:
                        print(f"❌ Error accediendo a {url_pelicula}: {e}")

                    # time.sleep(1)  # Espera para no saturar el servidor
        else:
            print(f"❌ Error al cargar la página {count_page}: {response.status_code}")

    except requests.exceptions.RequestException as e:
        print(f"❌ Error de conexión en la página {count_page}: {e}")

# Guardar datos
with open("peliculas_con_todos_los_datos_python.json", "w", encoding="utf-8") as f:
    json.dump(peliculas, f, indent=4, ensure_ascii=False)

print("✅ Datos guardados con todos los detalles en peliculas_con_todos_los_datos.json")
