# 🚀 HPE Analysis: CCA Region (Caucasus & Central Asia)

## 📌 Описание
Этот проект предназначен для автоматической обработки квартальных и годовых отчетов HPE Sell Track. 
Цель: фильтрация данных по 8 странам (CCA) и подготовка файлов для NotebookLM.

## 🛠 Доступные команды (Действия для ИИ)
1. **Обновить данные (Refresh)**: 
   - Найти последний `.xlsx` в `data/raw/`.
   - Использовать `scripts/sync_and_analyze.py`. Скрипт автоматически находит строку заголовка (ищет "Country") и сохраняет **все 100% колонок**, включая безымянные и сгруппированные.
   - Извлечь "Refresh Date" и создать CSV в `data/processed/`.
2. **Анализ изменений (Compare)**:
   - Автоматически выводится после обновления: разница в строках по странам и категориям.

## 🌍 Целевые страны (CCA-8)
Kazakhstan, Azerbaijan, Uzbekistan, Georgia, Armenia, Tajikistan, Kyrgyzstan, Turkmenistan.

## 💡 Технические нюансы
- **Column Integrity**: Запрещено удалять колонки, даже если у них нет заголовка ("Unnamed"). Они содержат данные по KPI.
- **Data Context**: Добавляется колонка `Source_Sheet`, чтобы NotebookLM понимал бизнес-юнит.

## 📂 Структура
- `data/raw/`: Исходные файлы от HPE.
- `data/processed/`: Готовые отчеты для NotebookLM.
- `scripts/`: Логика обработки.
