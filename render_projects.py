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

async def render_card(json_path: Path, artwork_path: Path, output_path: Path, browser):
    with open(json_path, encoding="utf-8") as f:
        card_data = json.load(f)

    logging.info(f"Processing card: {card_data['name']}")

    card_data["art"] = str(artwork_path.resolve())

    temp_json = json_path.parent / (json_path.stem + "_temp.json")
    with open(temp_json, "w", encoding="utf-8") as f:
        json.dump(card_data, f, indent=2)

    page = await browser.new_page()
    await page.set_viewport_size({"width": 1920, "height": 1080})
    await page.goto("https://cardconjurer.app/", wait_until="networkidle")
    await page.wait_for_selector("text=Import/Save", timeout=15000)

    try:
        await import_card(card_data, page)

        await add_margin(page)

        await change_artwork(artwork_path, page)

        await remove_set_symbol(page)

        await download_card(output_path, page)

        await page.wait_for_timeout(1000)
    finally:
        await page.close()
        temp_json.unlink()


async def import_card(card_data, page):
    # Open "Import/Save" tab
    await page.click('h3.selectable.readable-background[onclick*="toggleCreatorTabs"][onclick*="import"]')
    await page.wait_for_timeout(300)
    await page.select_option('#autoFrame', value='Seventh')
    await page.wait_for_timeout(500)
    await check_import_all_prints(page)
    await load_card(card_data, page)


async def load_card(card_data, page):
    # After selecting the frame: fill card name input
    await page.fill('#import-name', card_data['name'])
    await page.wait_for_timeout(200)
    # Send tab key to trigger onchange
    await page.keyboard.press('Tab')
    await page.wait_for_timeout(200)
    # After card name: select correct card version in dropdown
    card_version = f"{card_data['name']} ({card_data['set'].upper()} #{card_data['collector_number']})"
    # Find the matching option in the dropdown
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
        logging.warning(f"No matching card version found: {card_version}")


async def download_card(output_path, page):
    # --- Download the card ---
    async with page.expect_download() as download_info:
        await page.click('h3.download.padding[onclick="downloadCard();"]')
    download = await download_info.value
    await download.save_as(str(output_path))
    logging.info(f"Card saved: {output_path}")


async def add_margin(page):
    # After import: activate 'Frame' tab
    await page.click('h3.selectable.readable-background[onclick*="toggleCreatorTabs"][onclick*="frame"]')
    await page.wait_for_timeout(300)
    # In dropdown #selectFrameGroup select option with value 'margin'
    await page.select_option('#selectFrameGroup', value='Margin')
    await page.wait_for_timeout(300)
    # Click 'Add Frame to Card' button
    await page.click('#addToFull')
    await page.wait_for_timeout(300)


async def change_artwork(artwork_path, page):
    # --- Change artwork ---
    await page.click('h3.selectable.readable-background[onclick*="toggleCreatorTabs"][onclick*="art"]')
    await page.wait_for_timeout(500)
    art_input = await page.query_selector('input[type="file"][accept*=".png"][data-dropfunction="uploadArt"]')
    if art_input:
        await art_input.set_input_files(str(artwork_path))
        await page.wait_for_timeout(1500)
    else:
        logging.warning("Artwork input not found, image not changed.")

async def remove_set_symbol(page):
    # --- Remove set symbol ---
    await page.click('h3.selectable.readable-background[onclick*="toggleCreatorTabs"][onclick*="setSymbol"]')
    await page.wait_for_timeout(500)
    remove_btn = await page.query_selector('button.input.margin-bottom[onclick*="uploadSetSymbol(blank.src);"]')
    if remove_btn:
        await remove_btn.click()
        await page.wait_for_timeout(500)
    else:
        logging.warning("Remove Set Symbol button not found.")


async def check_import_all_prints(page):
    # Enable 'importAllPrints' checkbox if not already enabled
    checkbox = await page.query_selector('#importAllPrints')
    if checkbox:
        checked = await checkbox.is_checked()
        if not checked:
            parent = await checkbox.evaluate_handle('el => el.parentElement')
            await parent.click()
            await page.wait_for_timeout(200)


async def process_project(project_dir: Path, browser):
    input_dir = project_dir / "input"
    output_dir = project_dir / "output"
    output_dir.mkdir(exist_ok=True)

    json_files = sorted(input_dir.glob("*.json"))
    for json_file in json_files:
        base_name = json_file.stem
        artwork_file = input_dir / f"{base_name}.png"
        output_file = output_dir / f"{base_name}.png"

        if not artwork_file.exists():
            logging.warning(f"No artwork for: {base_name}")
            continue

        await render_card(json_file, artwork_file, output_file, browser)


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
                logging.info(f"Processing project: {project_name}")
                await process_project(project_dir, browser)
            else:
                logging.error(f"Project '{project_name}' not found in {BASE_DIR}")
        else:
            for project_dir in BASE_DIR.iterdir():
                if project_dir.is_dir():
                    logging.info(f"\nProcessing project: {project_dir.name}")
                    await process_project(project_dir, browser)

        await browser.close()

asyncio.run(main())
