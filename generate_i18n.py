import os
import json
import re

os.makedirs('locales', exist_ok=True)

langs = ['pt', 'en', 'es', 'fr', 'de', 'ru', 'zh']

dict_pt = {
    "title": "Canivete da Mata",
    "subtitle": "Sua ferramenta offline, leve e rápida para processamento de imagens e PDFs.",
    "remove_bg_title": "🪄 Remover Fundo",
    "remove_bg_desc": "Remove fundos brancos de imagens. Ajuste a tolerância para melhor precisão.",
    "tolerance": "Tolerância",
    "process": "Processar Transparência",
    "processing": "Processando...",
    "download": "⬇️ Baixar Arquivo",
    "close": "Fechar",
    "crop_title": "✂️ Recorte Quadrado Interativo",
    "crop_desc": "Selecione exatamente a parte da imagem que deseja recortar.",
    "size_original": "Tamanho Original do Recorte",
    "crop_btn": "Recortar Imagem",
    "resize_title": "📏 Redimensionar",
    "resize_desc": "Altere a resolução da sua imagem com precisão (Largura x Altura).",
    "width_px": "Largura (px)",
    "height_px": "Altura (px)",
    "resize_btn": "Redimensionar",
    "convert_title": "🔄 Converter Formato",
    "convert_desc": "Converta imagens entre PNG, JPG e BMP com controle de compressão.",
    "png_transp": "PNG (Transparente)",
    "jpg_comp": "JPG (Comprimido)",
    "bmp_lossless": "BMP (Sem perda)",
    "quality": "Qualidade",
    "convert_btn": "Converter",
    "img2pdf_title": "📄 Imagens p/ PDF",
    "img2pdf_desc": "Combine várias imagens em um único arquivo PDF.",
    "gen_pdf": "Gerar PDF",
    "svg2img_title": "🔤 SVG p/ Imagem",
    "svg2img_desc": "Renderize um vetor SVG para PNG puro.",
    "desired_width": "Largura Desejada em Pixels (Opcional):",
    "blank_original": "Deixe em branco para manter original",
    "rasterize_png": "Rasterizar para PNG",
    "pdf2img_title": "🖼️ PDF p/ Imagens (Extrair/Converter)",
    "pdf2img_desc": "Extraia imagens ou converta todas as páginas do PDF em imagens (em ZIP).",
    "desired_action": "Ação Desejada:",
    "convert_pages": "Converter as páginas inteiras em Imagem",
    "extract_images": "Apenas Extrair Imagens soltas contidas no PDF",
    "process_pdf": "Processar PDF",
    "compress_title": "📦 Compressor de Imagens",
    "compress_desc": "Reduza drasticamente o peso (MB/KB) de fotos JPG.",
    "compress_btn": "Comprimir Foto",
    "palette_title": "🎨 Paleta de Cores",
    "palette_desc": "Extraia as cores principais de uma imagem e veja os códigos HEX.",
    "extract_palette": "Extrair Paleta",
    "qr_gen_title": "📱 Gerar QR Code",
    "qr_gen_desc": "Crie um QR Code nítido a partir de qualquer texto ou link.",
    "qr_gen_ph": "Cole a URL ou digite algo...",
    "gen_qr_btn": "Gerar Imagem QR",
    "qr_read_title": "🔍 Ler QR Code",
    "qr_read_desc": "Envie a imagem de um QR Code ou código de barras e descubra o texto.",
    "decode_btn": "Decodificar Imagem",
    "pdf_merge_title": "📑 Juntar PDFs (Merge)",
    "pdf_merge_desc": "Selecione vários arquivos PDF e una-os em um único documento.",
    "merge_btn": "Unir PDFs",
    "pdf_split_title": "✂️ Dividir PDF (Split)",
    "pdf_split_desc": "Separe um documento PDF grande em páginas individuais (arquivo ZIP).",
    "split_btn": "Dividir PDF",
    "pdf_opt_title": "📉 Otimizar PDF",
    "pdf_opt_desc": "Comprima o tamanho interno de um PDF (remove fontes e metadados inúteis).",
    "opt_btn": "Otimizar Tamanho",
    "pdf_prot_title": "🔒 Proteger PDF (Senha)",
    "pdf_prot_desc": "Adicione uma senha ao seu PDF para proteger o documento.",
    "pass_ph": "Digite a Senha Forte",
    "protect_btn": "Proteger com Senha",
    "lang_select": "Idioma"
}

dict_en = {
    "title": "Jungle Knife",
    "subtitle": "Your lightweight, offline, and fast image and PDF processing tool.",
    "remove_bg_title": "🪄 Remove Background",
    "remove_bg_desc": "Remove white backgrounds from images. Adjust tolerance for better accuracy.",
    "tolerance": "Tolerance",
    "process": "Process Transparency",
    "processing": "Processing...",
    "download": "⬇️ Download File",
    "close": "Close",
    "crop_title": "✂️ Interactive Square Crop",
    "crop_desc": "Select exactly the part of the image you want to crop.",
    "size_original": "Original Crop Size",
    "crop_btn": "Crop Image",
    "resize_title": "📏 Resize",
    "resize_desc": "Change the resolution of your image precisely (Width x Height).",
    "width_px": "Width (px)",
    "height_px": "Height (px)",
    "resize_btn": "Resize",
    "convert_title": "🔄 Convert Format",
    "convert_desc": "Convert images between PNG, JPG, and BMP with compression control.",
    "png_transp": "PNG (Transparent)",
    "jpg_comp": "JPG (Compressed)",
    "bmp_lossless": "BMP (Lossless)",
    "quality": "Quality",
    "convert_btn": "Convert",
    "img2pdf_title": "📄 Images to PDF",
    "img2pdf_desc": "Combine multiple images into a single PDF file.",
    "gen_pdf": "Generate PDF",
    "svg2img_title": "🔤 SVG to Image",
    "svg2img_desc": "Render an SVG vector to pure PNG.",
    "desired_width": "Desired Width in Pixels (Optional):",
    "blank_original": "Leave blank to keep original",
    "rasterize_png": "Rasterize to PNG",
    "pdf2img_title": "🖼️ PDF to Images (Extract/Convert)",
    "pdf2img_desc": "Extract images or convert all pages of a PDF to images (in ZIP).",
    "desired_action": "Desired Action:",
    "convert_pages": "Convert full pages to Image",
    "extract_images": "Only extract isolated images from PDF",
    "process_pdf": "Process PDF",
    "compress_title": "📦 Image Compressor",
    "compress_desc": "Drastically reduce the size (MB/KB) of JPG photos.",
    "compress_btn": "Compress Photo",
    "palette_title": "🎨 Color Palette",
    "palette_desc": "Extract the main colors from an image and see HEX codes.",
    "extract_palette": "Extract Palette",
    "qr_gen_title": "📱 Generate QR Code",
    "qr_gen_desc": "Create a sharp QR Code from any text or link.",
    "qr_gen_ph": "Paste URL or type something...",
    "gen_qr_btn": "Generate QR Image",
    "qr_read_title": "🔍 Read QR Code",
    "qr_read_desc": "Upload a QR Code or barcode image to reveal the text.",
    "decode_btn": "Decode Image",
    "pdf_merge_title": "📑 Merge PDFs",
    "pdf_merge_desc": "Select multiple PDF files and combine them into a single document.",
    "merge_btn": "Merge PDFs",
    "pdf_split_title": "✂️ Split PDF",
    "pdf_split_desc": "Separate a large PDF document into individual pages (ZIP file).",
    "split_btn": "Split PDF",
    "pdf_opt_title": "📉 Optimize PDF",
    "pdf_opt_desc": "Compress the internal size of a PDF (removes useless fonts and metadata).",
    "opt_btn": "Optimize Size",
    "pdf_prot_title": "🔒 Protect PDF (Password)",
    "pdf_prot_desc": "Add a password to your PDF to protect the document.",
    "pass_ph": "Enter Strong Password",
    "protect_btn": "Protect with Password",
    "lang_select": "Language"
}

dict_es = {
    "title": "Navaja de la Selva",
    "subtitle": "Tu herramienta offline, ligera y rápida para procesamiento de imágenes y PDF.",
    "remove_bg_title": "🪄 Quitar Fondo",
    "remove_bg_desc": "Elimina fondos blancos de imágenes. Ajusta la tolerancia para mayor precisión.",
    "tolerance": "Tolerancia",
    "process": "Procesar Transparencia",
    "processing": "Procesando...",
    "download": "⬇️ Descargar Archivo",
    "close": "Cerrar",
    "crop_title": "✂️ Recorte Cuadrado Interactivo",
    "crop_desc": "Selecciona exactamente la parte de la imagen que deseas recortar.",
    "size_original": "Tamaño Original del Recorte",
    "crop_btn": "Recortar Imagen",
    "resize_title": "📏 Redimensionar",
    "resize_desc": "Cambia la resolución de tu imagen con precisión (Ancho x Alto).",
    "width_px": "Ancho (px)",
    "height_px": "Alto (px)",
    "resize_btn": "Redimensionar",
    "convert_title": "🔄 Convertir Formato",
    "convert_desc": "Convierte imágenes entre PNG, JPG y BMP con control de compresión.",
    "png_transp": "PNG (Transparente)",
    "jpg_comp": "JPG (Comprimido)",
    "bmp_lossless": "BMP (Sin pérdida)",
    "quality": "Calidad",
    "convert_btn": "Convertir",
    "img2pdf_title": "📄 Imágenes a PDF",
    "img2pdf_desc": "Combina varias imágenes en un solo archivo PDF.",
    "gen_pdf": "Generar PDF",
    "svg2img_title": "🔤 SVG a Imagen",
    "svg2img_desc": "Renderiza un vector SVG a PNG puro.",
    "desired_width": "Ancho Deseado en Píxeles (Opcional):",
    "blank_original": "Dejar en blanco para mantener original",
    "rasterize_png": "Rasterizar a PNG",
    "pdf2img_title": "🖼️ PDF a Imágenes (Extraer/Convertir)",
    "pdf2img_desc": "Extrae imágenes o convierte todas las páginas del PDF a imágenes (en ZIP).",
    "desired_action": "Acción Deseada:",
    "convert_pages": "Convertir las páginas enteras a Imagen",
    "extract_images": "Solo extraer imágenes aisladas del PDF",
    "process_pdf": "Procesar PDF",
    "compress_title": "📦 Compresor de Imágenes",
    "compress_desc": "Reduce drásticamente el peso (MB/KB) de fotos JPG.",
    "compress_btn": "Comprimir Foto",
    "palette_title": "🎨 Paleta de Colores",
    "palette_desc": "Extrae los colores principales de una imagen y mira los códigos HEX.",
    "extract_palette": "Extraer Paleta",
    "qr_gen_title": "📱 Generar Código QR",
    "qr_gen_desc": "Crea un Código QR nítido desde cualquier texto o enlace.",
    "qr_gen_ph": "Pega la URL o escribe algo...",
    "gen_qr_btn": "Generar Imagen QR",
    "qr_read_title": "🔍 Leer Código QR",
    "qr_read_desc": "Sube la imagen de un Código QR o código de barras y descubre el texto.",
    "decode_btn": "Decodificar Imagen",
    "pdf_merge_title": "📑 Unir PDFs (Merge)",
    "pdf_merge_desc": "Selecciona varios archivos PDF y únelos en un solo documento.",
    "merge_btn": "Unir PDFs",
    "pdf_split_title": "✂️ Dividir PDF (Split)",
    "pdf_split_desc": "Separa un documento PDF grande en páginas individuales (archivo ZIP).",
    "split_btn": "Dividir PDF",
    "pdf_opt_title": "📉 Optimizar PDF",
    "pdf_opt_desc": "Comprime el tamaño interno de un PDF (elimina fuentes y metadatos inútiles).",
    "opt_btn": "Optimizar Tamaño",
    "pdf_prot_title": "🔒 Proteger PDF (Contraseña)",
    "pdf_prot_desc": "Agrega una contraseña a tu PDF para proteger el documento.",
    "pass_ph": "Escribe una Contraseña Segura",
    "protect_btn": "Proteger con Contraseña",
    "lang_select": "Idioma"
}

dict_fr = {
    "title": "Couteau de la Jungle",
    "subtitle": "Votre outil hors ligne, léger et rapide pour le traitement d'images et de PDF.",
    "remove_bg_title": "🪄 Supprimer l'arrière-plan",
    "remove_bg_desc": "Supprimez les arrière-plans blancs des images. Ajustez la tolérance pour une meilleure précision.",
    "tolerance": "Tolérance",
    "process": "Traiter la Transparence",
    "processing": "Traitement...",
    "download": "⬇️ Télécharger le fichier",
    "close": "Fermer",
    "crop_title": "✂️ Recadrage Carré Interactif",
    "crop_desc": "Sélectionnez exactement la partie de l'image que vous souhaitez recadrer.",
    "size_original": "Taille de recadrage d'origine",
    "crop_btn": "Recadrer l'image",
    "resize_title": "📏 Redimensionner",
    "resize_desc": "Modifiez la résolution de votre image avec précision (Largeur x Hauteur).",
    "width_px": "Largeur (px)",
    "height_px": "Hauteur (px)",
    "resize_btn": "Redimensionner",
    "convert_title": "🔄 Convertir Format",
    "convert_desc": "Convertissez des images entre PNG, JPG et BMP avec contrôle de compression.",
    "png_transp": "PNG (Transparent)",
    "jpg_comp": "JPG (Compressé)",
    "bmp_lossless": "BMP (Sans perte)",
    "quality": "Qualité",
    "convert_btn": "Convertir",
    "img2pdf_title": "📄 Images en PDF",
    "img2pdf_desc": "Combinez plusieurs images en un seul fichier PDF.",
    "gen_pdf": "Générer PDF",
    "svg2img_title": "🔤 SVG en Image",
    "svg2img_desc": "Restituez un vecteur SVG en PNG pur.",
    "desired_width": "Largeur souhaitée en pixels (facultatif):",
    "blank_original": "Laissez vide pour conserver l'original",
    "rasterize_png": "Pixelliser en PNG",
    "pdf2img_title": "🖼️ PDF en Images (Extraire/Convertir)",
    "pdf2img_desc": "Extrayez des images ou convertissez toutes les pages du PDF en images (en ZIP).",
    "desired_action": "Action souhaitée:",
    "convert_pages": "Convertir des pages entières en Image",
    "extract_images": "Uniquement extraire les images isolées du PDF",
    "process_pdf": "Traiter le PDF",
    "compress_title": "📦 Compresseur d'images",
    "compress_desc": "Réduisez drastiquement le poids (Mo/Ko) des photos JPG.",
    "compress_btn": "Compresser Photo",
    "palette_title": "🎨 Palette de Couleurs",
    "palette_desc": "Extrayez les couleurs principales d'une image et voyez les codes HEX.",
    "extract_palette": "Extraire Palette",
    "qr_gen_title": "📱 Générer QR Code",
    "qr_gen_desc": "Créez un QR Code net à partir de n'importe quel texte ou lien.",
    "qr_gen_ph": "Collez l'URL ou tapez quelque chose...",
    "gen_qr_btn": "Générer Image QR",
    "qr_read_title": "🔍 Lire QR Code",
    "qr_read_desc": "Téléchargez une image QR Code ou code-barres et découvrez le texte.",
    "decode_btn": "Décoder l'image",
    "pdf_merge_title": "📑 Fusionner PDFs",
    "pdf_merge_desc": "Sélectionnez plusieurs fichiers PDF et combinez-les en un seul document.",
    "merge_btn": "Unir PDFs",
    "pdf_split_title": "✂️ Diviser PDF",
    "pdf_split_desc": "Séparez un grand document PDF en pages individuelles (fichier ZIP).",
    "split_btn": "Diviser PDF",
    "pdf_opt_title": "📉 Optimiser PDF",
    "pdf_opt_desc": "Compressez la taille interne d'un PDF (supprime les polices et métadonnées inutiles).",
    "opt_btn": "Optimiser la Taille",
    "pdf_prot_title": "🔒 Protéger PDF (Mot de passe)",
    "pdf_prot_desc": "Ajoutez un mot de passe à votre PDF pour protéger le document.",
    "pass_ph": "Entrez un mot de passe fort",
    "protect_btn": "Protéger avec mot de passe",
    "lang_select": "Langue"
}

dict_de = {
    "title": "Dschungelmesser",
    "subtitle": "Ihr offline, leichtes und schnelles Tool zur Bild- und PDF-Verarbeitung.",
    "remove_bg_title": "🪄 Hintergrund entfernen",
    "remove_bg_desc": "Entfernen Sie weiße Hintergründe aus Bildern. Passen Sie die Toleranz für bessere Genauigkeit an.",
    "tolerance": "Toleranz",
    "process": "Transparenz verarbeiten",
    "processing": "Wird bearbeitet...",
    "download": "⬇️ Datei herunterladen",
    "close": "Schließen",
    "crop_title": "✂️ Interaktives quadratisches Zuschneiden",
    "crop_desc": "Wählen Sie genau den Teil des Bildes aus, den Sie zuschneiden möchten.",
    "size_original": "Ursprüngliche Schnittgröße",
    "crop_btn": "Bild zuschneiden",
    "resize_title": "📏 Größe ändern",
    "resize_desc": "Ändern Sie die Auflösung Ihres Bildes präzise (Breite x Höhe).",
    "width_px": "Breite (px)",
    "height_px": "Höhe (px)",
    "resize_btn": "Größe ändern",
    "convert_title": "🔄 Format konvertieren",
    "convert_desc": "Konvertieren Sie Bilder zwischen PNG, JPG und BMP mit Komprimierungssteuerung.",
    "png_transp": "PNG (Transparent)",
    "jpg_comp": "JPG (Komprimiert)",
    "bmp_lossless": "BMP (Verlustfrei)",
    "quality": "Qualität",
    "convert_btn": "Konvertieren",
    "img2pdf_title": "📄 Bilder in PDF",
    "img2pdf_desc": "Kombinieren Sie mehrere Bilder in einer einzigen PDF-Datei.",
    "gen_pdf": "PDF generieren",
    "svg2img_title": "🔤 SVG in Bild",
    "svg2img_desc": "Rendern Sie einen SVG-Vektor zu reinem PNG.",
    "desired_width": "Gewünschte Breite in Pixel (Optional):",
    "blank_original": "Leer lassen, um Original beizubehalten",
    "rasterize_png": "In PNG rastern",
    "pdf2img_title": "🖼️ PDF in Bilder (Extrahieren/Konvertieren)",
    "pdf2img_desc": "Extrahieren Sie Bilder oder konvertieren Sie alle Seiten der PDF in Bilder (in ZIP).",
    "desired_action": "Gewünschte Aktion:",
    "convert_pages": "Ganze Seiten in Bilder konvertieren",
    "extract_images": "Nur isolierte Bilder aus PDF extrahieren",
    "process_pdf": "PDF verarbeiten",
    "compress_title": "📦 Bildkompressor",
    "compress_desc": "Reduzieren Sie das Gewicht (MB/KB) von JPG-Fotos drastisch.",
    "compress_btn": "Foto komprimieren",
    "palette_title": "🎨 Farbpalette",
    "palette_desc": "Extrahieren Sie die Hauptfarben aus einem Bild und sehen Sie die HEX-Codes.",
    "extract_palette": "Palette extrahieren",
    "qr_gen_title": "📱 QR-Code generieren",
    "qr_gen_desc": "Erstellen Sie einen scharfen QR-Code aus jedem Text oder Link.",
    "qr_gen_ph": "URL einfügen oder etwas eingeben...",
    "gen_qr_btn": "QR-Bild generieren",
    "qr_read_title": "🔍 QR-Code lesen",
    "qr_read_desc": "Laden Sie ein QR-Code- oder Barcode-Bild hoch und decken Sie den Text auf.",
    "decode_btn": "Bild dekodieren",
    "pdf_merge_title": "📑 PDFs zusammenführen",
    "pdf_merge_desc": "Wählen Sie mehrere PDF-Dateien aus und kombinieren Sie sie in einem einzigen Dokument.",
    "merge_btn": "PDFs zusammenführen",
    "pdf_split_title": "✂️ PDF teilen",
    "pdf_split_desc": "Trennen Sie ein großes PDF-Dokument in einzelne Seiten (ZIP-Datei).",
    "split_btn": "PDF teilen",
    "pdf_opt_title": "📉 PDF optimieren",
    "pdf_opt_desc": "Komprimieren Sie die interne Größe einer PDF (entfernt nutzlose Schriftarten und Metadaten).",
    "opt_btn": "Größe optimieren",
    "pdf_prot_title": "🔒 PDF schützen (Passwort)",
    "pdf_prot_desc": "Fügen Sie Ihrer PDF ein Passwort hinzu, um das Dokument zu schützen.",
    "pass_ph": "Starkes Passwort eingeben",
    "protect_btn": "Mit Passwort schützen",
    "lang_select": "Sprache"
}

dict_ru = {
    "title": "Нож для джунглей",
    "subtitle": "Ваш легкий, автономный и быстрый инструмент для обработки изображений и PDF.",
    "remove_bg_title": "🪄 Удалить фон",
    "remove_bg_desc": "Удаляет белый фон с изображений. Настройте допуск для большей точности.",
    "tolerance": "Допуск",
    "process": "Обработать прозрачность",
    "processing": "Обработка...",
    "download": "⬇️ Скачать файл",
    "close": "Закрыть",
    "crop_title": "✂️ Интерактивная квадратная обрезка",
    "crop_desc": "Выберите именно ту часть изображения, которую хотите обрезать.",
    "size_original": "Оригинальный размер обрезки",
    "crop_btn": "Обрезать изображение",
    "resize_title": "📏 Изменить размер",
    "resize_desc": "Точно измените разрешение изображения (Ширина x Высота).",
    "width_px": "Ширина (px)",
    "height_px": "Высота (px)",
    "resize_btn": "Изменить размер",
    "convert_title": "🔄 Преобразовать формат",
    "convert_desc": "Преобразуйте изображения между PNG, JPG и BMP с контролем сжатия.",
    "png_transp": "PNG (Прозрачный)",
    "jpg_comp": "JPG (Сжатый)",
    "bmp_lossless": "BMP (Без потерь)",
    "quality": "Качество",
    "convert_btn": "Конвертировать",
    "img2pdf_title": "📄 Изображения в PDF",
    "img2pdf_desc": "Объедините несколько изображений в один PDF файл.",
    "gen_pdf": "Сгенерировать PDF",
    "svg2img_title": "🔤 SVG в изображение",
    "svg2img_desc": "Отрендерите векторный SVG в чистый PNG.",
    "desired_width": "Желаемая ширина в пикселях (необязательно):",
    "blank_original": "Оставьте пустым для сохранения оригинала",
    "rasterize_png": "Растеризовать в PNG",
    "pdf2img_title": "🖼️ PDF в изображения (Извлечь/Конверт)",
    "pdf2img_desc": "Извлеките изображения или конвертируйте все страницы PDF в изображения (в ZIP).",
    "desired_action": "Желаемое действие:",
    "convert_pages": "Конвертировать целые страницы в изображение",
    "extract_images": "Извлекать только изолированные изображения из PDF",
    "process_pdf": "Обработать PDF",
    "compress_title": "📦 Компрессор изображений",
    "compress_desc": "Значительно уменьшите вес (МБ/КБ) JPG фотографий.",
    "compress_btn": "Сжать фото",
    "palette_title": "🎨 Цветовая палитра",
    "palette_desc": "Извлеките основные цвета из изображения и посмотрите HEX-коды.",
    "extract_palette": "Извлечь палитру",
    "qr_gen_title": "📱 Создать QR-код",
    "qr_gen_desc": "Создайте четкий QR-код из любого текста или ссылки.",
    "qr_gen_ph": "Вставьте URL или введите что-то...",
    "gen_qr_btn": "Сгенерировать QR-изображение",
    "qr_read_title": "🔍 Читать QR-код",
    "qr_read_desc": "Загрузите изображение QR-кода или штрих-кода, чтобы узнать текст.",
    "decode_btn": "Декодировать изображение",
    "pdf_merge_title": "📑 Объединить PDF",
    "pdf_merge_desc": "Выберите несколько файлов PDF и объедините их в один документ.",
    "merge_btn": "Объединить PDF",
    "pdf_split_title": "✂️ Разделить PDF",
    "pdf_split_desc": "Разделите большой PDF-документ на отдельные страницы (ZIP файл).",
    "split_btn": "Разделить PDF",
    "pdf_opt_title": "📉 Оптимизировать PDF",
    "pdf_opt_desc": "Сожмите внутренний размер PDF (удаляет бесполезные шрифты и метаданные).",
    "opt_btn": "Оптимизировать размер",
    "pdf_prot_title": "🔒 Защитить PDF (Пароль)",
    "pdf_prot_desc": "Добавьте пароль к вашему PDF, чтобы защитить документ.",
    "pass_ph": "Введите надежный пароль",
    "protect_btn": "Защитить паролем",
    "lang_select": "Язык"
}

dict_zh = {
    "title": "丛林刀",
    "subtitle": "您的离线、轻量级、快速的图像和 PDF 处理工具。",
    "remove_bg_title": "🪄 移除背景",
    "remove_bg_desc": "从图像中删除白色背景。调整容差以获得更好的准确性。",
    "tolerance": "容差",
    "process": "处理透明度",
    "processing": "处理中...",
    "download": "⬇️ 下载文件",
    "close": "关闭",
    "crop_title": "✂️ 交互式方形裁剪",
    "crop_desc": "准确选择您想要裁剪的图像部分。",
    "size_original": "原始裁剪尺寸",
    "crop_btn": "裁剪图像",
    "resize_title": "📏 调整大小",
    "resize_desc": "精确更改图像的分辨率（宽 x 高）。",
    "width_px": "宽度 (px)",
    "height_px": "高度 (px)",
    "resize_btn": "调整大小",
    "convert_title": "🔄 转换格式",
    "convert_desc": "在 PNG、JPG 和 BMP 之间转换图像，并控制压缩。",
    "png_transp": "PNG（透明）",
    "jpg_comp": "JPG（压缩）",
    "bmp_lossless": "BMP（无损）",
    "quality": "质量",
    "convert_btn": "转换",
    "img2pdf_title": "📄 图像转 PDF",
    "img2pdf_desc": "将多张图像合并为一个 PDF 文件。",
    "gen_pdf": "生成 PDF",
    "svg2img_title": "🔤 SVG 转图像",
    "svg2img_desc": "将 SVG 矢量渲染为纯 PNG。",
    "desired_width": "期望宽度（像素，可选）：",
    "blank_original": "留空保持原样",
    "rasterize_png": "栅格化为 PNG",
    "pdf2img_title": "🖼️ PDF 转图像（提取/转换）",
    "pdf2img_desc": "提取图像或将 PDF 的所有页面转换为图像（ZIP 格式）。",
    "desired_action": "期望操作：",
    "convert_pages": "将整个页面转换为图像",
    "extract_images": "仅从 PDF 中提取隔离的图像",
    "process_pdf": "处理 PDF",
    "compress_title": "📦 图像压缩器",
    "compress_desc": "大幅减少 JPG 照片的体积 (MB/KB)。",
    "compress_btn": "压缩照片",
    "palette_title": "🎨 调色板",
    "palette_desc": "从图像中提取主要颜色并查看 HEX 代码。",
    "extract_palette": "提取调色板",
    "qr_gen_title": "📱 生成二维码",
    "qr_gen_desc": "从任何文本或链接创建清晰的二维码。",
    "qr_gen_ph": "粘贴 URL 或输入内容...",
    "gen_qr_btn": "生成二维码图像",
    "qr_read_title": "🔍 读取二维码",
    "qr_read_desc": "上传二维码或条形码图像以发现文本。",
    "decode_btn": "解码图像",
    "pdf_merge_title": "📑 合并 PDF",
    "pdf_merge_desc": "选择多个 PDF 文件并将它们合并为一个文档。",
    "merge_btn": "合并 PDF",
    "pdf_split_title": "✂️ 拆分 PDF",
    "pdf_split_desc": "将大型 PDF 文档分隔成单页（ZIP 文件）。",
    "split_btn": "拆分 PDF",
    "pdf_opt_title": "📉 优化 PDF",
    "pdf_opt_desc": "压缩 PDF 的内部大小（删除无用的字体和元数据）。",
    "opt_btn": "优化大小",
    "pdf_prot_title": "🔒 保护 PDF（密码）",
    "pdf_prot_desc": "为您的 PDF 添加密码以保护文档。",
    "pass_ph": "输入强密码",
    "protect_btn": "使用密码保护",
    "lang_select": "语言"
}

all_dicts = {
    'pt': dict_pt, 'en': dict_en, 'es': dict_es,
    'fr': dict_fr, 'de': dict_de, 'ru': dict_ru, 'zh': dict_zh
}

for lang, d in all_dicts.items():
    with open(f'locales/{lang}.json', 'w') as f:
        json.dump(d, f, ensure_ascii=False, indent=2)

with open('templates/index.html', 'r') as f:
    html = f.read()

# Substituir textos puros por {{ .T "key" }}
# Title/Subtitle
html = html.replace('Canivete da Mata - Ferramentas de Imagem e PDF', '{{ .T "title" }} - Tools')
html = html.replace('<h1>Canivete da Mata</h1>', '<h1>{{ .T "title" }}</h1>')
html = html.replace('<p>Sua ferramenta offline, leve e rápida para processamento de imagens e PDFs.</p>', '<p>{{ .T "subtitle" }}</p>')

# Rem BG
html = html.replace('🪄 Remover Fundo', '{{ .T "remove_bg_title" }}')
html = html.replace('Remove fundos brancos de imagens. Ajuste a tolerância para melhor precisão.', '{{ .T "remove_bg_desc" }}')
html = html.replace('Tolerância (Branco):', '{{ .T "tolerance" }}:')
html = html.replace('Processar Transparência', '{{ .T "process" }}')

# Shared UI
html = html.replace('Processando...', '{{ .T "processing" }}')
html = html.replace('⬇️ Baixar Arquivo', '{{ .T "download" }}')
html = html.replace('Fechar</button>', '{{ .T "close" }}</button>')

# Crop
html = html.replace('✂️ Recorte Quadrado Interativo', '{{ .T "crop_title" }}')
html = html.replace('Selecione exatamente a parte da imagem que deseja recortar.', '{{ .T "crop_desc" }}')
html = html.replace('Tamanho Original do Recorte', '{{ .T "size_original" }}')
html = html.replace('Recortar Imagem', '{{ .T "crop_btn" }}')

# Resize
html = html.replace('📏 Redimensionar', '{{ .T "resize_title" }}')
html = html.replace('Altere a resolução da sua imagem com precisão (Largura x Altura).', '{{ .T "resize_desc" }}')
html = html.replace('Largura (px)', '{{ .T "width_px" }}')
html = html.replace('Altura (px)', '{{ .T "height_px" }}')
html = html.replace('>Redimensionar</button>', '>{{ .T "resize_btn" }}</button>')

# Convert
html = html.replace('🔄 Converter Formato', '{{ .T "convert_title" }}')
html = html.replace('Converta imagens entre PNG, JPG e BMP com controle de compressão.', '{{ .T "convert_desc" }}')
html = html.replace('PNG (Transparente)', '{{ .T "png_transp" }}')
html = html.replace('JPG (Comprimido)', '{{ .T "jpg_comp" }}')
html = html.replace('BMP (Sem perda)', '{{ .T "bmp_lossless" }}')
html = html.replace('Qualidade JPG:', '{{ .T "quality" }}:')
html = html.replace('>Converter</button>', '>{{ .T "convert_btn" }}</button>')

# Img2Pdf
html = html.replace('📄 Imagens p/ PDF', '{{ .T "img2pdf_title" }}')
html = html.replace('Combine várias imagens em um único arquivo PDF.', '{{ .T "img2pdf_desc" }}')
html = html.replace('Gerar PDF', '{{ .T "gen_pdf" }}')

# SVG2Img
html = html.replace('🔤 SVG p/ Imagem', '{{ .T "svg2img_title" }}')
html = html.replace('Renderize um vetor SVG para PNG puro.', '{{ .T "svg2img_desc" }}')
html = html.replace('Largura Desejada em Pixels (Opcional):', '{{ .T "desired_width" }}')
html = html.replace('Deixe em branco para manter original', '{{ .T "blank_original" }}')
html = html.replace('Rasterizar para PNG', '{{ .T "rasterize_png" }}')

# Pdf2Img
html = html.replace('🖼️ PDF p/ Imagens (Extrair/Converter)', '{{ .T "pdf2img_title" }}')
html = html.replace('Extraia imagens ou converta todas as páginas do PDF em imagens (em ZIP).', '{{ .T "pdf2img_desc" }}')
html = html.replace('Ação Desejada:', '{{ .T "desired_action" }}')
html = html.replace('Converter as páginas inteiras em Imagem', '{{ .T "convert_pages" }}')
html = html.replace('Apenas Extrair Imagens soltas contidas no PDF', '{{ .T "extract_images" }}')
html = html.replace('Processar PDF', '{{ .T "process_pdf" }}')

# Compress
html = html.replace('📦 Compressor de Imagens', '{{ .T "compress_title" }}')
html = html.replace('Reduza drasticamente o peso (MB/KB) de fotos JPG.', '{{ .T "compress_desc" }}')
html = html.replace('Comprimir Foto', '{{ .T "compress_btn" }}')
html = html.replace('Qualidade:', '{{ .T "quality" }}:')

# Palette
html = html.replace('🎨 Paleta de Cores', '{{ .T "palette_title" }}')
html = html.replace('Extraia as cores principais de uma imagem e veja os códigos HEX.', '{{ .T "palette_desc" }}')
html = html.replace('Extrair Paleta', '{{ .T "extract_palette" }}')

# QR Gen
html = html.replace('📱 Gerar QR Code', '{{ .T "qr_gen_title" }}')
html = html.replace('Crie um QR Code nítido a partir de qualquer texto ou link.', '{{ .T "qr_gen_desc" }}')
html = html.replace('Cole a URL ou digite algo...', '{{ .T "qr_gen_ph" }}')
html = html.replace('Gerar Imagem QR', '{{ .T "gen_qr_btn" }}')

# QR Read
html = html.replace('🔍 Ler QR Code', '{{ .T "qr_read_title" }}')
html = html.replace('Envie a imagem de um QR Code ou código de barras e descubra o texto.', '{{ .T "qr_read_desc" }}')
html = html.replace('Decodificar Imagem', '{{ .T "decode_btn" }}')

# Merge
html = html.replace('📑 Juntar PDFs (Merge)', '{{ .T "pdf_merge_title" }}')
html = html.replace('Selecione vários arquivos PDF e una-os em um único documento.', '{{ .T "pdf_merge_desc" }}')
html = html.replace('Unir PDFs', '{{ .T "merge_btn" }}')

# Split
html = html.replace('✂️ Dividir PDF (Split)', '{{ .T "pdf_split_title" }}')
html = html.replace('Separe um documento PDF grande em páginas individuais (arquivo ZIP).', '{{ .T "pdf_split_desc" }}')
html = html.replace('Dividir PDF', '{{ .T "split_btn" }}')

# Optimize
html = html.replace('📉 Otimizar PDF', '{{ .T "pdf_opt_title" }}')
html = html.replace('Comprima o tamanho interno de um PDF (remove fontes e metadados inúteis).', '{{ .T "pdf_opt_desc" }}')
html = html.replace('Otimizar Tamanho', '{{ .T "opt_btn" }}')

# Protect
html = html.replace('🔒 Proteger PDF (Senha)', '{{ .T "pdf_prot_title" }}')
html = html.replace('Adicione uma senha ao seu PDF para proteger o documento.', '{{ .T "pdf_prot_desc" }}')
html = html.replace('Digite a Senha Forte', '{{ .T "pass_ph" }}')
html = html.replace('Proteger com Senha', '{{ .T "protect_btn" }}')

lang_picker = """        <div style="position:absolute; top:1rem; right:1rem; text-align:right;">
            <select id="lang-select" onchange="changeLang(this.value)" style="width: auto; padding: 5px 30px 5px 10px;">
                <option value="pt" {{ if eq .Lang "pt" }}selected{{ end }}>🇧🇷 PT</option>
                <option value="en" {{ if eq .Lang "en" }}selected{{ end }}>🇺🇸 EN</option>
                <option value="es" {{ if eq .Lang "es" }}selected{{ end }}>🇪🇸 ES</option>
                <option value="fr" {{ if eq .Lang "fr" }}selected{{ end }}>🇫🇷 FR</option>
                <option value="de" {{ if eq .Lang "de" }}selected{{ end }}>🇩🇪 DE</option>
                <option value="ru" {{ if eq .Lang "ru" }}selected{{ end }}>🇷🇺 RU</option>
                <option value="zh" {{ if eq .Lang "zh" }}selected{{ end }}>🇨🇳 ZH</option>
            </select>
        </div>
"""

# Insert Language Picker at the top of container
html = html.replace('<main class="container">', f'<main class="container">\n{lang_picker}')

js_change_lang = """
        function changeLang(lang) {
            document.cookie = "lang=" + lang + ";path=/;max-age=31536000";
            window.location.reload();
        }
"""
html = html.replace('</script>', f'{js_change_lang}</script>')

with open('templates/index.html', 'w') as f:
    f.write(html)
