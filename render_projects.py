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

class CardConjurerAutomation:
    def __init__(self, project_dir: Path):
        self.project_dir = project_dir
        self.input_dir = project_dir / "input"
        self.output_dir = project_dir / "output"
        self.output_dir.mkdir(exist_ok=True)
        self.browser = None

    async def run(self):
        async with async_playwright() as p:
            self.browser = await p.chromium.launch(
                headless=False,
                #args=["--start-maximized"]
            )
            logging.info(f"Processing project: {self.project_dir}")
            await self.process_project()
            await self.browser.close()

    async def process_project(self):
        json_files = sorted(self.input_dir.glob("*.json"))
        for json_file in json_files:
            base_name = json_file.stem
            artwork_file = self.input_dir / f"{base_name}.png"
            output_file = self.output_dir / f"{base_name}.png"

            if not artwork_file.exists():
                logging.warning(f"No artwork for: {base_name}")
                continue

            await self.render_card(json_file, artwork_file, output_file)

    async def render_card(self, json_path: Path, artwork_path: Path, output_path: Path):
        with open(json_path, encoding="utf-8") as f:
            card_data = json.load(f)

        logging.info(f"Processing card: {card_data['name']}")

        card_data["art"] = str(artwork_path.resolve())

        temp_json = json_path.parent / (json_path.stem + "_temp.json")
        with open(temp_json, "w", encoding="utf-8") as f:
            json.dump(card_data, f, indent=2)

        page = await self.browser.new_page()
        await page.set_viewport_size({"width": 1920, "height": 1080})
        await page.goto("https://cardconjurer.app/", wait_until="networkidle")
        await page.wait_for_selector("text=Import/Save", timeout=15000)

        try:
            await self.import_card(card_data, page)
            await self.add_margin(page)
            await self.change_artwork(artwork_path, page)
            await self.remove_set_symbol(page)
            await self.download_card(output_path, page)
            await page.wait_for_timeout(1000)
        finally:
            await page.close()
            temp_json.unlink()

    async def import_card(self, card_data, page):
        await page.click('h3.selectable.readable-background[onclick*="toggleCreatorTabs"][onclick*="import"]')
        await page.wait_for_timeout(300)
        await page.select_option('#autoFrame', value='Seventh')
        await page.wait_for_timeout(500)
        await self.check_import_all_prints(page)
        await self.load_card(card_data, page)

    async def load_card(self, card_data, page):
        await page.fill('#import-name', card_data['name'])
        await page.wait_for_timeout(200)
        await page.keyboard.press('Tab')
        await page.wait_for_timeout(200)
        card_version = f"{card_data['name']} ({card_data['set'].upper()} #{card_data['collector_number']})"
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

    async def download_card(self, output_path, page):
        async with page.expect_download() as download_info:
            await page.click('h3.download.padding[onclick="downloadCard();"]')
        download = await download_info.value
        await download.save_as(str(output_path))
        logging.info(f"Card saved: {output_path}")

    async def add_margin(self, page):
        await page.click('h3.selectable.readable-background[onclick*="toggleCreatorTabs"][onclick*="frame"]')
        await page.wait_for_timeout(300)
        await page.select_option('#selectFrameGroup', value='Margin')
        await page.wait_for_timeout(300)
        await page.click('#addToFull')
        await page.wait_for_timeout(300)

    async def change_artwork(self, artwork_path, page):
        await page.click('h3.selectable.readable-background[onclick*="toggleCreatorTabs"][onclick*="art"]')
        await page.wait_for_timeout(500)
        art_input = await page.query_selector('input[type="file"][accept*=".png"][data-dropfunction="uploadArt"]')
        if art_input:
            await art_input.set_input_files(str(artwork_path))
            await page.wait_for_timeout(1500)
        else:
            logging.warning("Artwork input not found, image not changed.")

    async def remove_set_symbol(self, page):
        await page.click('h3.selectable.readable-background[onclick*="toggleCreatorTabs"][onclick*="setSymbol"]')
        await page.wait_for_timeout(500)
        remove_btn = await page.query_selector('button.input.margin-bottom[onclick*="uploadSetSymbol(blank.src);"]')
        if remove_btn:
            await remove_btn.click()
            await page.wait_for_timeout(500)
        else:
            logging.warning("Remove Set Symbol button not found.")

    async def check_import_all_prints(self, page):
        checkbox = await page.query_selector('#importAllPrints')
        if checkbox:
            checked = await checkbox.is_checked()
            if not checked:
                parent = await checkbox.evaluate_handle('el => el.parentElement')
                await parent.click()
                await page.wait_for_timeout(200)

def main():
    if len(sys.argv) < 2:
        logging.error("Please provide the project folder as a command-line argument.")
        return

    project_dir = Path(sys.argv[1])
    if not project_dir.exists() or not project_dir.is_dir():
        logging.error(f"Project folder '{project_dir}' not found or is not a directory.")
        return

    automation = CardConjurerAutomation(project_dir)
    asyncio.run(automation.run())

if __name__ == "__main__":
    main()
