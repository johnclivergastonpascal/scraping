import requests
from bs4 import BeautifulSoup
import json
import time

total_pages = 1
series = []

# Crear sesión para mantener la conexión abierta
session = requests.Session()
session.headers.update({
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36"
})

def obtener_detalle(url_pelicula):
    try:
        detalle_res = session.get(url_pelicula)
        if detalle_res.status_code == 200:
            return BeautifulSoup(detalle_res.text, "html.parser")
        else:
            print(f"⚠️ Error accediendo a detalle de la serie: {url_pelicula}")
            return None
    except requests.exceptions.RequestException as e:
        print(f"❌ Error de conexión al detalle de la serie {url_pelicula}: {e}")
        return None

for count_page in range(1, total_pages + 1):
    page = f"https://sololatino.net/series/page/{count_page}/"
    print(f"📄 Procesando página {count_page}...")

    try:
        response = session.get(page)
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
                    detalle_soup = obtener_detalle(url_pelicula)
                    if detalle_soup:
                        # Fondo
                        wallpaper_div = detalle_soup.find("div", class_="wallpaper")
                        wallpaper_url = "No disponible"
                        if wallpaper_div and "style" in wallpaper_div.attrs:
                            style = wallpaper_div["style"]
                            start = style.find("url(")
                            end = style.find(")", start)
                            if start != -1 and end != -1:
                                wallpaper_url = style[start + 4:end].strip()

                        # Puntuación
                        puntuacion_div = detalle_soup.find("div", class_="nota")
                        puntuacion = puntuacion_div.find("span").text.strip() if puntuacion_div and puntuacion_div.find("span") else "No disponible"

                        # Descripción y sinopsis
                        descripcion_div = detalle_soup.find("div", itemprop="description", class_="wp-content")
                        sinopsis = descripcion = "No disponible"
                        if descripcion_div:
                            sinopsis_tag = descripcion_div.find("h3")
                            sinopsis = sinopsis_tag.text.strip() if sinopsis_tag else "No disponible"
                            description_tag = descripcion_div.find("p")
                            descripcion = description_tag.text.strip() if description_tag else "No disponible"

                        imagen_url = img.get("data-srcset") if img else "No disponible"

                        # Director y reparto
                        director = {}
                        reparto = []

                        srepart_div = detalle_soup.find("div", class_="sbox srepart")
                        if srepart_div:
                            # Director
                            director_div = srepart_div.find("div", itemprop="director")
                            if director_div:
                                try:
                                    director = {
                                        "nombre": director_div.find("meta", itemprop="name")["content"].strip(),
                                        "url": director_div.find("a")["href"].strip(),
                                        "imagen": director_div.find("img")["src"].strip()
                                    }
                                except Exception as e:
                                    print(f"⚠️ Error procesando director: {e}")

                            # Reparto
                            actores_divs = srepart_div.find_all("div", class_="person", itemtype="http://schema.org/Person")
                            for actor_div in actores_divs:
                                try:
                                    nombre = actor_div.find("meta", itemprop="name")["content"]
                                    url = actor_div.find("a")["href"]
                                    imagen = actor_div.find("img")["src"]
                                    personaje = actor_div.find("div", class_="caracter").text.strip()

                                    reparto.append({
                                        "nombre": nombre,
                                        "url": url,
                                        "imagen": imagen,
                                        "personaje": personaje
                                    })
                                except Exception as e:
                                    print(f"⚠️ Error procesando actor: {e}")

                        # Temporadas con episodios
                        temporadas = []
                        temporadas_divs = detalle_soup.select("div.se-c[data-season]")

                        for temporada_div in temporadas_divs:
                            temporada_numero = temporada_div["data-season"]
                            episodios = []

                            episodios_ul = temporada_div.find("ul", class_="episodios")
                            if episodios_ul:
                                for li in episodios_ul.find_all("li"):
                                    a_tag = li.find("a")
                                    if a_tag:
                                        href = a_tag["href"]
                                        img_tag = a_tag.find("img")
                                        epst = a_tag.find("div", class_="epst")
                                        numerando = a_tag.find("div", class_="numerando")
                                        date = a_tag.find("span", class_="date")

                                        # Obtener sinopsis y video del episodio
                                        sinopsis_episodio = "No disponible"
                                        video_url_episodio = "No disponible"
                                        try:
                                            episodio_detalle = session.get(href)
                                            if episodio_detalle.status_code == 200:
                                                detalle_episodio_soup = BeautifulSoup(episodio_detalle.text, "html.parser")
                                                desc_div = detalle_episodio_soup.find("div", itemprop="description", class_="wp-content")
                                                if desc_div and desc_div.find("p"):
                                                    sinopsis_episodio = desc_div.find("p").text.strip()

                                                iframe = detalle_episodio_soup.find("iframe", class_="metaframe rptss")
                                                if iframe and iframe.has_attr("src"):
                                                    video_url_episodio = iframe["src"]
                                        except Exception as e:
                                            print(f"⚠️ No se pudo acceder al detalle del episodio: {href} - {e}")

                                        episodios.append({
                                            "titulo": epst.text.strip() if epst else "No disponible",
                                            "numero": numerando.text.strip() if numerando else "No disponible",
                                            "fecha": date.text.strip() if date else "No disponible",
                                            "imagen": img_tag["src"] if img_tag else "No disponible",
                                            "url": href,
                                            "sinopsis": sinopsis_episodio,
                                            "video_url": video_url_episodio
                                        })

                            temporadas.append({
                                "temporada": int(temporada_numero),
                                "episodios": episodios
                            })


                        series.append({
                            "year": year.text.strip(),
                            "titulo": title.text.strip(),
                            "genero": genero.text.strip(),
                            "imagen": imagen_url,
                            "url": url_pelicula,
                            "puntuacion": puntuacion,
                            "wallpaper": wallpaper_url,
                            "sinopsis": sinopsis,
                            "descripcion": descripcion,
                            "director": director,
                            "reparto": reparto,
                            "temporadas": temporadas,
                        })

                        print(f"🎬 Agregada: {title.text.strip()} - ⭐ {puntuacion}")
                    else:
                        print(f"⚠️ No se pudo acceder al detalle: {url_pelicula}")

        else:
            print(f"❌ Error al cargar la página {count_page}: {response.status_code}")
    except requests.exceptions.RequestException as e:
        print(f"❌ Error de conexión en la página {count_page}: {e}")

    time.sleep(1)  # Controlar el tiempo de espera para evitar sobrecargar el servidor

# Guardar datos
with open("series_con_todos_los_datos.json", "w", encoding="utf-8") as f:
    json.dump(series, f, indent=4, ensure_ascii=False)

print("✅ Datos guardados en series_con_todos_los_datos.json")
