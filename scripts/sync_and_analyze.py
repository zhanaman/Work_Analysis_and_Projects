import pandas as pd
import os
import glob
from datetime import datetime

PROJECT_ROOT = '/Users/anonimouskz/Documents/Projects/01_ACTIVE/HPE/Analysis_FY26'
RAW_DIR = os.path.join(PROJECT_ROOT, 'data/raw')
PROCESSED_DIR = os.path.join(PROJECT_ROOT, 'data/processed')

TARGET_COUNTRIES = [
    'Kazakhstan', 'Azerbaijan', 'Uzbekistan', 'Georgia', 
    'Armenia', 'Tajikistan', 'Kyrgyzstan', 'Turkmenistan'
]

def get_refresh_date(file_path):
    try:
        xl = pd.ExcelFile(file_path)
        if 'Networking' in xl.sheet_names:
            df = pd.read_excel(file_path, sheet_name='Networking', nrows=5)
            # Ищем ячейку с датой в Networking или Readme
            for col in df.columns:
                val = df[col].iloc[0]
                if isinstance(val, datetime):
                    return val.strftime('%Y-%m-%d')
            # Если не нашли, ищем колонку Refreshed date
            if 'Refreshed date' in df.columns:
                date_val = df['Refreshed date'].iloc[0]
                return str(date_val).split()[0]
    except:
        pass
    return datetime.now().strftime('%Y-%m-%d')

def process_file():
    files = glob.glob(os.path.join(RAW_DIR, '*.xlsx'))
    if not files:
        print("No .xlsx files found in data/raw/")
        return
    
    latest_raw = max(files, key=os.path.getmtime)
    refresh_date = get_refresh_date(latest_raw)
    output_name = f"HPE_CCA_{refresh_date}.csv"
    output_path = os.path.join(PROCESSED_DIR, output_name)
    
    print(f"Processing: {os.path.basename(latest_raw)}")
    xl = pd.ExcelFile(latest_raw)
    all_data = []
    
    for sheet in xl.sheet_names:
        if any(x in sheet.lower() for x in ['readme', 'information']): continue
        
        # Читаем лист полностью, чтобы найти заголовок
        full_df = pd.read_excel(latest_raw, sheet_name=sheet, header=None)
        
        # Ищем строку, где есть 'Country'
        header_idx = None
        for i, row in full_df.iterrows():
            if 'Country' in [str(v).strip() for v in row.values]:
                header_idx = i
                break
        
        if header_idx is None:
            print(f"  - {sheet}: Country column not found. Skipping.")
            continue

        # Перечитываем лист с правильным заголовком
        df = pd.read_excel(latest_raw, sheet_name=sheet, header=header_idx)
        
        # Ищем колонку Country (она может называться чуть иначе из-за пробелов)
        country_col = [c for c in df.columns if str(c).strip() == 'Country']
        if not country_col:
            continue
            
        col_name = country_col[0]
        df[col_name] = df[col_name].astype(str).str.strip()
        filtered = df[df[col_name].isin(TARGET_COUNTRIES)].copy()
        
        if not filtered.empty:
            filtered['Source_Sheet'] = sheet
            # Важно: Сохраняем АБСОЛЮТНО все колонки, не удаляя Unnamed
            all_data.append(filtered)
            print(f"  - {sheet}: Found {len(filtered)} rows, {len(df.columns)} columns")

    if all_data:
        # Объединяем листы. sort=False сохраняет порядок колонок
        final_df = pd.concat(all_data, ignore_index=True, sort=False)
        
        # Ставим Source_Sheet первым
        cols = ['Source_Sheet'] + [c for c in final_df.columns if c != 'Source_Sheet']
        final_df = final_df[cols]
        
        # Сохраняем с поддержкой длинных строк и спецсимволов
        final_df.to_csv(output_path, index=False, encoding='utf-8-sig')
        print(f"\nSUCCESS: Created {output_name}")
        print(f"Total rows: {len(final_df)} | Total unique columns: {len(final_df.columns)}")
        
        # Выводим человеческий отчет
        compare_with_previous(final_df, output_name)
    else:
        print("ERROR: No data found.")

def compare_with_previous(new_df, current_name):
    prev_files = [f for f in glob.glob(os.path.join(PROCESSED_DIR, 'HPE_CCA_*.csv')) if os.path.basename(f) != current_name]
    if not prev_files: return
    
    old_file = max(prev_files, key=os.path.getmtime)
    old_df = pd.read_csv(old_file, low_memory=False)
    
    print(f"\n--- Разница с предыдущим файлом ({os.path.basename(old_file)}) ---")
    diff = len(new_df) - len(old_df)
    print(f"Всего строк: {len(new_df)} ({'+' if diff >= 0 else ''}{diff})")
    
    # По странам
    new_counts = new_df['Country'].value_counts()
    old_counts = old_df['Country'].value_counts()
    for c in TARGET_COUNTRIES:
        n, o = new_counts.get(c, 0), old_counts.get(c, 0)
        if n != o:
            print(f"  - {c}: {o} -> {n} ({'+' if n-o > 0 else ''}{n-o})")

if __name__ == '__main__':
    process_file()
