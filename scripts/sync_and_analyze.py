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
            if 'Refreshed date' in df.columns:
                date_val = df['Refreshed date'].iloc[0]
                if isinstance(date_val, datetime):
                    return date_val.strftime('%Y-%m-%d')
                return str(date_val).split()[0]
    except Exception:
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
    print(f"Detected Refresh Date: {refresh_date}")
    
    xl = pd.ExcelFile(latest_raw)
    all_data = []
    
    for sheet in xl.sheet_names:
        if any(x in sheet.lower() for x in ['readme', 'information']): continue
        
        header_check = pd.read_excel(latest_raw, sheet_name=sheet, nrows=20, header=None)
        header_row = 0
        for i, row in header_check.iterrows():
            if 'Country' in row.values:
                header_row = i
                break
        
        df = pd.read_excel(latest_raw, sheet_name=sheet, skiprows=header_row)
        df = df.loc[:, ~df.columns.astype(str).str.contains('^Unnamed')]
        
        if 'Country' in df.columns:
            df['Country'] = df['Country'].astype(str).str.strip()
            filtered = df[df['Country'].isin(TARGET_COUNTRIES)].copy()
            if not filtered.empty:
                filtered['Source_Sheet'] = sheet
                all_data.append(filtered)
    
    if all_data:
        new_df = pd.concat(all_data, ignore_index=True)
        new_df.to_csv(output_path, index=False, encoding='utf-8-sig')
        print(f"Saved to: {output_name}")
        compare_with_previous(new_df, output_name)
    else:
        print("No CCA data found.")

def compare_with_previous(new_df, current_name):
    existing_files = glob.glob(os.path.join(PROCESSED_DIR, 'HPE_CCA_*.csv'))
    # Исключаем текущий файл из списка для сравнения
    prev_files = [f for f in existing_files if os.path.basename(f) != current_name]
    
    if not prev_files:
        print("\n--- Initial version created. ---")
        return
    
    old_file = max(prev_files, key=os.path.getmtime)
    old_df = pd.read_csv(old_file)
    
    print(f"\n--- Comparison with {os.path.basename(old_file)} ---")
    
    diff = len(new_df) - len(old_df)
    if diff > 0:
        print(f"📈 Добавлено {diff} новых записей.")
    elif diff < 0:
        print(f"📉 Удалено {abs(diff)} записей.")
    else:
        print("⚖️ Количественных изменений нет.")
    
    # Детальный анализ по категориям (Source_Sheet)
    print("\nДетализация по категориям:")
    for sheet in new_df['Source_Sheet'].unique():
        n_sheet = len(new_df[new_df['Source_Sheet'] == sheet])
        o_sheet = len(old_df[old_df['Source_Sheet'] == sheet]) if 'Source_Sheet' in old_df.columns else 0
        if n_sheet != o_sheet:
            diff_s = n_sheet - o_sheet
            print(f"  - {sheet}: {'+' if diff_s > 0 else ''}{diff_s} записей (всего {n_sheet})")

    # Краткая аналитика по странам
    print("\nИзменения по странам:")
    new_counts = new_df['Country'].value_counts()
    old_counts = old_df['Country'].value_counts()
    
    for c in TARGET_COUNTRIES:
        n, o = new_counts.get(c, 0), old_counts.get(c, 0)
        if n != o:
            diff_c = n - o
            print(f"  - {c}: {'+' if diff_c > 0 else ''}{diff_c} ({o} -> {n})")

if __name__ == '__main__':
    process_file()
