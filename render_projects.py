import asyncio
from pathlib import Path
from playwright.async_api import async_playwright
import json
import sys
import logging
import argparse
import csv
import xml.etree.ElementTree as ET
import uuid

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%H:%M:%S"
)

class CardConjurerAutomation:
    def __init__(self, csv_path: Path, input_dir: Path, output_dir: Path, card_names=None, skip_images=False):
        self.csv_path = csv_path
        self.input_dir = input_dir
        self.output_dir = output_dir
        self.output_dir.mkdir(exist_ok=True)
        self.browser = None
        self.card_names = set(card_names) if card_names else None
        self.generated_cards = []
        self.skip_images = skip_images

    async def run(self):
        if self.skip_images:
            logging.info("Skipping image generation, only writing XML.")
            await self.process_csv(skip_images=True)
            return
        async with async_playwright() as p:
            self.browser = await p.chromium.launch(
                headless=False,
                #args=["--start-maximized"]
            )
            logging.info(f"Processing cards from: {self.csv_path}")
            await self.process_csv()
            await self.browser.close()

    async def process_csv(self, skip_images=False):
        with open(self.csv_path, newline='', encoding='utf-8') as csvfile:
            reader = csv.reader(csvfile)
            for row in reader:
                if len(row) < 4:
                    logging.warning(f"Skipping invalid row: {row}")
                    continue
                count, card_name, set_code, collector_number = row
                count = int(count) if count.isdigit() else 1

                # Only process if card_name is in the filter (if set)
                if self.card_names and card_name not in self.card_names:
                    continue

                artwork_file = self.input_dir / f"{card_name}_{set_code}_{collector_number}.png"
                if not artwork_file.exists():
                    logging.warning(f"No artwork for: {card_name}_{set_code}_{collector_number}, using default image.")
                    artwork_file = None

                card_data = {
                    "name": card_name,
                    "set": set_code,
                    "collector_number": collector_number
                }

                # Replace spaces with underscores in output file name
                safe_card_name = card_name.replace(" ", "_")
                safe_set_code = set_code.replace(" ", "_")
                safe_collector_number = collector_number.replace(" ", "_")

                for i in range(count):
                    output_file = self.output_dir / f"{safe_card_name}_{safe_set_code}_{safe_collector_number}_{i+1:04}.png"
                    if not skip_images:
                        await self.render_card(card_data, artwork_file, output_file)
                    # Track generated card for XML
                    self.generated_cards.append({
                        "name": output_file.name,
                        "query": card_name.lower()
                    })

    async def render_card(self, card_data, artwork_path, output_path):
        logging.info(f"Processing card: {card_data['name']} ({card_data['set']} #{card_data['collector_number']})")

        page = await self.browser.new_page()
        await page.set_viewport_size({"width": 1920, "height": 1080})
        await page.goto("https://cardconjurer.app/", wait_until="networkidle")
        await page.wait_for_selector("text=Import/Save", timeout=15000)

        try:
            await self.import_card(card_data, page)
            await self.add_margin(page)
            if artwork_path:
                await self.change_artwork(artwork_path, page)
            await self.remove_set_symbol(page)
            await self.download_card(output_path, page)
            await page.wait_for_timeout(1000)
        finally:
            await page.close()

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
        await page.wait_for_timeout(1000)
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
            await page.wait_for_timeout(2000)
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

    def get_bracket(self, quantity):
        brackets = [
            18, 36, 55, 72, 90, 108, 126, 144, 162, 180, 198, 216, 234, 396, 504, 612
        ]
        for b in brackets:
            if quantity <= b:
                return b
        return brackets[-1]

    def write_mpc_xml(self, xml_path):
        order = ET.Element("order")
        details = ET.SubElement(order, "details")
        quantity = len(self.generated_cards)
        bracket = self.get_bracket(quantity)
        ET.SubElement(details, "quantity").text = str(quantity)
        ET.SubElement(details, "bracket").text = str(bracket)
        ET.SubElement(details, "stock").text = "(S30) Standard Smooth"
        ET.SubElement(details, "foil").text = "false"

        fronts = ET.SubElement(order, "fronts")
        for idx, card in enumerate(self.generated_cards):
            card_elem = ET.SubElement(fronts, "card")
            ET.SubElement(card_elem, "id").text = str(uuid.uuid4())
            ET.SubElement(card_elem, "slots").text = str(idx)
            ET.SubElement(card_elem, "name").text = card["name"]
            ET.SubElement(card_elem, "query").text = card["query"]

        ET.SubElement(order, "backs")
        ET.SubElement(order, "cardback")

        tree = ET.ElementTree(order)
        tree.write(xml_path, encoding="utf-8", xml_declaration=True)

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("csv_file", help="Path to the CSV file")
    parser.add_argument("--artworks", help="Input folder for artwork images", default="input")
    parser.add_argument("--output", help="Output folder for generated cards", default="output")
    parser.add_argument("--cards", help="Comma-separated list of card names to process", default=None)
    parser.add_argument("--skip-images", action="store_true", help="Only write the XML file, do not generate images")
    args = parser.parse_args()

    csv_path = Path(args.csv_file)
    input_dir = Path(args.artworks)
    output_dir = Path(args.output)
    card_names = [name.strip() for name in args.cards.split(",")] if args.cards else None

    if not csv_path.exists() or not csv_path.is_file():
        logging.error(f"CSV file '{csv_path}' not found or is not a file.")
        return

    automation = CardConjurerAutomation(csv_path, input_dir, output_dir, card_names, skip_images=args.skip_images)
    asyncio.run(automation.run())
    # Write XML after processing, use the same name as the csv file with .xml
    xml_path = output_dir / (csv_path.stem + ".xml")
    automation.write_mpc_xml(xml_path)
    logging.info(f"XML file written to: {xml_path}")

if __name__ == "__main__":
    main()
