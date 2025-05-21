import asyncio
from pathlib import Path
from playwright.async_api import async_playwright
import json
import sys
import logging

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%H:%M:%S"
)

SCRIPT_DIR = Path(__file__).parent.resolve()
BASE_DIR = SCRIPT_DIR / "projects"

async def render_card(json_path: Path, artwork_path: Path, output_path: Path, browser, frame_path: Path):
    with open(json_path, encoding="utf-8") as f:
        card_data = json.load(f)

    logging.info(f"🔍 Processing card: {card_data['name']}")

    card_data["art"] = str(artwork_path.resolve())

    temp_json = json_path.parent / (json_path.stem + "_temp.json")
    with open(temp_json, "w", encoding="utf-8") as f:
        json.dump(card_data, f, indent=2)

    page = await browser.new_page()
    await page.set_viewport_size({"width": 1920, "height": 1080})
    await page.goto("https://cardconjurer.app/", wait_until="networkidle")
    await page.wait_for_selector("text=Import/Save", timeout=15000)

    try:
        # Import frame file
        await page.click("text=Import/Save")
        await page.wait_for_timeout(500)
        # Suche gezielt das Input-Element für .cardconjurer/.txt
        frame_file_input = await page.query_selector('input[type="file"][accept=".cardconjurer,.txt"]')
        if not frame_file_input:
            logging.error("❌ Spezifisches Frame file input nicht gefunden.")
            return

        if not frame_path.exists():
            logging.error(f"❌ Frame file not found: {frame_path}")
            return

        #await load_card_frame(frame_file_input, frame_path, page)

        # Danach 'Seventh Edition' im autoFrame-Dropdown auswählen
        await page.select_option('#autoFrame', value='Seventh')
        await page.wait_for_timeout(500)

        await check_import_all_prints(page)

        await load_card(card_data, page)

        await add_margin(page)

        await change_artwork(artwork_path, page)

        await download_card(output_path, page)

        await page.wait_for_timeout(1000)
    finally:
        await page.close()
        temp_json.unlink()


async def load_card(card_data, page):
    # Nach Auswahl des Frames: Kartennamen in das Eingabefeld eintragen
    await page.fill('#import-name', card_data['name'])
    await page.wait_for_timeout(200)
    # Sende Tab-Taste, damit onchange ausgelöst wird
    await page.keyboard.press('Tab')
    await page.wait_for_timeout(200)
    # Nach Kartennamen: Korrekte Karten-Version im Dropdown auswählen
    # Erzeuge das Suchmuster wie im Dropdown: 'Sakura-Tribe Elder (TDC #266)'
    card_version = f"{card_data['name']} ({card_data['set'].upper()} #{card_data['collector_number']})"
    # Hole alle Optionen und finde die passende
    options = await page.query_selector_all('#import-index option')
    value_to_select = None
    for option in options:
        text = await option.text_content()
        if text == card_version:
            value_to_select = await option.get_attribute('value')
            break
    if value_to_select is not None:
        await page.select_option('#import-index', value=value_to_select)
        await page.wait_for_timeout(500)
    else:
        logging.warning(f"⚠️ Keine passende Karten-Version gefunden: {card_version}")


async def download_card(output_path, page):
    # --- Download der Karte ---
    async with page.expect_download() as download_info:
        await page.click('h3.download.padding[onclick="downloadCard();"]')
    download = await download_info.value
    await download.save_as(str(output_path))
    logging.info(f"💾 Karte gespeichert: {output_path}")


async def add_margin(page):
    # Nach Import: 'Frame'-Tab aktivieren
    await page.click('h3.selectable.readable-background[onclick*="toggleCreatorTabs"][onclick*="frame"]')
    await page.wait_for_timeout(1000)
    # Im Dropdown #selectFrameGroup die Option mit value 'margin' auswählen
    await page.select_option('#selectFrameGroup', value='Margin')
    await page.wait_for_timeout(1000)
    # Button 'Add Frame to Card' klicken
    await page.click('#addToFull')
    await page.wait_for_timeout(1000)


async def change_artwork(artwork_path, page):
    # --- NEU: Artwork ändern ---
    # Zum "Art"-Tab wechseln
    await page.click('h3.selectable.readable-background[onclick*="toggleCreatorTabs"][onclick*="art"]')
    await page.wait_for_timeout(500)
    # Das passende File-Input finden und Artwork hochladen
    art_input = await page.query_selector('input[type="file"][accept*=".png"][data-dropfunction="uploadArt"]')
    if art_input:
        await art_input.set_input_files(str(artwork_path))
        await page.wait_for_timeout(1500)
    else:
        logging.warning("⚠️ Artwork-Input nicht gefunden, Bild nicht geändert.")


async def load_card_frame(frame_file_input, frame_path, page):
    await frame_file_input.set_input_files(str(frame_path))
    await page.wait_for_timeout(1000)
    logging.info(f"🖼️ Frame import versucht: {frame_path}")
    # Nach dem Import: Auswahl im Dropdown setzen
    await page.select_option('#load-card-options', label='red_frame')
    await page.wait_for_timeout(1000)
    logging.info("✅ Frame import erfolgreich.")


async def check_import_all_prints(page):
    # Checkbox 'importAllPrints' aktivieren, falls nicht bereits aktiviert
    checkbox = await page.query_selector('#importAllPrints')
    if checkbox:
        logging.info("✅ Checkbox 'importAllPrints' gefunden.")
        checked = await checkbox.is_checked()
        logging.info(f"✅ Checkbox 'importAllPrints' ist {'aktiviert' if checked else 'deaktiviert'}.")
        if not checked:
            parent = await checkbox.evaluate_handle('el => el.parentElement')
            await parent.click()

            # await checkbox.check()
            await page.wait_for_timeout(200)


async def process_project(project_dir: Path, browser):
    input_dir = project_dir / "input"
    output_dir = project_dir / "output"
    output_dir.mkdir(exist_ok=True)

    frames_dir = project_dir / "frames"
    # Passe ggf. den Frame-Dateinamen an, falls er variabel ist
    frame_files = list(frames_dir.glob("*.cardconjurer"))
    if not frame_files:
        logging.warning(f"⚠️ No frame file found in: {frames_dir}")
        return
    frame_path = frame_files[0]

    json_files = sorted(input_dir.glob("*.json"))
    for json_file in json_files:
        base_name = json_file.stem
        artwork_file = input_dir / f"{base_name}.png"
        output_file = output_dir / f"{base_name}.png"

        if not artwork_file.exists():
            logging.warning(f"⚠️ No artwork for: {base_name}")
            continue

        await render_card(json_file, artwork_file, output_file, browser, frame_path)


async def main():
    # Project name as argument (optional)
    project_name = sys.argv[1] if len(sys.argv) > 1 else None

    async with async_playwright() as p:
        browser = await p.chromium.launch(
            headless=False,
            args=["--start-maximized"]
        )

        if project_name:
            project_dir = BASE_DIR / project_name
            if project_dir.exists() and project_dir.is_dir():
                logging.info(f"🔧 Processing project: {project_name}")
                await process_project(project_dir, browser)
            else:
                logging.error(f"❌ Project '{project_name}' not found in {BASE_DIR}")
        else:
            for project_dir in BASE_DIR.iterdir():
                if project_dir.is_dir():
                    logging.info(f"\n🔧 Project: {project_dir.name}")
                    await process_project(project_dir, browser)

        await browser.close()

asyncio.run(main())
