import pandas as pd
import os
import glob
from datetime import datetime

PROJECT_ROOT = '/Users/anonimouskz/Documents/Projects/01_ACTIVE/HPE/Analysis_FY26'
RAW_DIR = os.path.join(PROJECT_ROOT, 'data/raw')
PROCESSED_DIR = os.path.join(PROJECT_ROOT, 'data/processed')
RAD_FILE = '/Users/anonimouskz/Desktop/RAD_to_be_analized.csv'

TARGET_COUNTRIES = [
    'Kazakhstan', 'Azerbaijan', 'Uzbekistan', 'Georgia', 
    'Armenia', 'Tajikistan', 'Kyrgyzstan', 'Turkmenistan'
]

AI_FOCUS_PARTNERS = [
    'FREEDOM TELECOM', 'QAZCLOUD', 'BESTCOMP', 'EURODESIGN', 'ULTRA', 
    'SMART TECH SOLUTION', 'IT ACTIV', 'KAZAKHTELECOM', 'SOFTLINE'
]

def get_refresh_date(file_path):
    try:
        xl = pd.ExcelFile(file_path)
        if 'Networking' in xl.sheet_names:
            df = pd.read_excel(file_path, sheet_name='Networking', nrows=5)
            for col in df.columns:
                val = df[col].iloc[0]
                if isinstance(val, datetime): return val.strftime('%Y-%m-%d')
    except: pass
    return datetime.now().strftime('%Y-%m-%d')

def load_rad_data():
    if os.path.exists(RAD_FILE):
        try:
            rad_df = pd.read_csv(RAD_FILE, sep=None, engine='python', on_bad_lines='skip')
            return rad_df[['Partner Name', 'RAD']].drop_duplicates()
        except: pass
    return None

def process_file():
    files = glob.glob(os.path.join(RAW_DIR, '*.xlsx'))
    if not files: return print("No .xlsx files found.")
    
    latest_raw = max(files, key=os.path.getmtime)
    refresh_date = get_refresh_date(latest_raw)
    
    # Создаем папку для текущей выгрузки
    batch_dir = os.path.join(PROCESSED_DIR, f"Analysis_{refresh_date}")
    os.makedirs(batch_dir, exist_ok=True)
    
    rad_map = load_rad_data()
    xl = pd.ExcelFile(latest_raw)
    total_saved = 0
    
    print(f"Processing: {os.path.basename(latest_raw)}")
    print(f"Output directory: {batch_dir}\n")

    for sheet in xl.sheet_names:
        # Обрабатываем лист с метаданными отдельно
        if 'column level information' in sheet.lower():
            meta_df = pd.read_excel(latest_raw, sheet_name=sheet)
            meta_path = os.path.join(batch_dir, "Metadata_Column_Information.csv")
            meta_df.to_csv(meta_path, index=False, encoding='utf-8-sig')
            print(f"  - {sheet}: Saved as metadata")
            continue

        if any(x in sheet.lower() for x in ['readme', 'information']): continue
        
        # Читаем весь лист, чтобы найти строку с заголовками
        full_df = pd.read_excel(latest_raw, sheet_name=sheet, header=None)
        header_idx = next((i for i, row in full_df.iterrows() if 'Country' in [str(v).strip() for v in row.values]), None)
        
        if header_idx is None: continue
        
        # Читаем данные с правильным заголовком. pandas сохранит все колонки, включая Unnamed.
        df = pd.read_excel(latest_raw, sheet_name=sheet, header=header_idx)
        
        country_col = [c for c in df.columns if str(c).strip() == 'Country'][0]
        partner_col_candidates = [c for c in df.columns if any(x in str(c).upper() for x in ['PARTY NAME', 'PARTNER NAME', 'ACCOUNT NAME'])]
        partner_col = partner_col_candidates[0] if partner_col_candidates else None
        
        df[country_col] = df[country_col].astype(str).str.strip()
        
        # Фильтруем данные по целевым странам
        sheet_filtered = df[df[country_col].isin(TARGET_COUNTRIES)].copy()
        
        if not sheet_filtered.empty:
            # Разделяем данные листа по странам
            for country in sheet_filtered[country_col].unique():
                country_df = sheet_filtered[sheet_filtered[country_col] == country].copy()
                
                # Добавляем бизнес-контекст
                country_df['Source_Sheet'] = sheet
                
                if partner_col:
                    if rad_map is not None:
                        country_df = country_df.merge(rad_map, left_on=partner_col, right_on='Partner Name', how='left').drop(columns=['Partner Name'], errors='ignore')
                    
                    country_df['Zenith_AI_Potential'] = country_df[partner_col].apply(
                        lambda x: 'High' if any(ai.lower() in str(x).upper() for ai in AI_FOCUS_PARTNERS) else 'Standard'
                    )
                
                # Формируем имя файла: Sheet_Country.csv (убираем пробелы)
                clean_sheet = sheet.replace(" ", "_")
                clean_country = country.replace(" ", "_")
                file_name = f"{clean_sheet}_{clean_country}.csv"
                file_path = os.path.join(batch_dir, file_name)
                
                # Сохраняем важные колонки в начало
                base_cols = ['Source_Sheet', 'RAD', 'Zenith_AI_Potential']
                cols = [c for c in base_cols if c in country_df.columns] + [c for c in country_df.columns if c not in base_cols]
                
                country_df[cols].to_csv(file_path, index=False, encoding='utf-8-sig')
                total_saved += 1
            
            print(f"  - {sheet}: Processed {len(sheet_filtered)} rows for {sheet_filtered[country_col].nunique()} countries")

    if total_saved > 0:
        print(f"\nSUCCESS: Created {total_saved} files in {batch_dir}")
    else:
        print("ERROR: No data found matching criteria.")

if __name__ == '__main__':
    process_file()
