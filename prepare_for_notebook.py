import pandas as pd
import os

file_path = 'data/FY26_PR_EMEA_Sell_track.xlsx'
output_file = 'HPE_CCA_Analysis_Ready.csv'

target_countries = [
    'Kazakhstan', 'Azerbaijan', 'Uzbekistan', 'Georgia', 
    'Armenia', 'Tajikistan', 'Kyrgyzstan', 'Turkmenistan'
]

exclude_sheets = ['Readme', 'Column level information']

all_data = []

xl = pd.ExcelFile(file_path)

for sheet in xl.sheet_names:
    if sheet in exclude_sheets:
        continue
    
    print(f"Processing sheet: {sheet}...")
    
    header_check = pd.read_excel(file_path, sheet_name=sheet, nrows=20, header=None)
    header_row = None
    
    for i, row in header_check.iterrows():
        if 'Country' in row.values:
            header_row = i
            break
            
    if header_row is None:
        print(f"  Warning: 'Country' column not found in sheet {sheet}. Skipping.")
        continue

    df = pd.read_excel(file_path, sheet_name=sheet, skiprows=header_row)
    df = df.loc[:, ~df.columns.astype(str).str.contains('^Unnamed')]
    
    if 'Country' in df.columns:
        df['Country'] = df['Country'].astype(str).str.strip()
        filtered_df = df[df['Country'].isin(target_countries)].copy()
        
        if not filtered_df.empty:
            filtered_df['Source_Sheet'] = sheet
            all_data.append(filtered_df)
            print(f"  Success: Found {len(filtered_df)} rows for CCA.")
        else:
            print(f"  Info: No CCA countries found in {sheet}.")

if all_data:
    final_df = pd.concat(all_data, ignore_index=True)
    cols = ['Source_Sheet'] + [c for c in final_df.columns if c != 'Source_Sheet']
    final_df = final_df[cols]
    final_df.to_csv(output_file, index=False, encoding='utf-8-sig')
    print(f"\nDONE! File saved as: {output_file}")
    print(f"Total rows collected: {len(final_df)}")
else:
    print("\nERROR: No data collected.")
