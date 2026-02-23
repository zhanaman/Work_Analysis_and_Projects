# 🚀 HPE Analysis: CCA Region (Caucasus & Central Asia)

## 📌 Описание
Этот проект предназначен для автоматической обработки квартальных и годовых отчетов HPE Sell Track. 
Цель: фильтрация данных по 8 странам (CCA) и подготовка файлов для NotebookLM.

## 🛠 Доступные команды (Действия для ИИ)
1. **Обновить данные (Refresh)**: 
   - Найти последний `.xlsx` в `data/raw/`.
   - Извлечь "Refresh Date" из листа Readme.
   - Создать новый CSV в `data/processed/` с именем `HPE_CCA_[Date].csv`.
2. **Анализ изменений (Compare)**:
   - Сравнить последний созданный CSV с предыдущей версией.
   - Вывести человекочитаемый отчет: сколько строк добавилось, изменились ли KPI по странам.

## 🌍 Целевые страны (CCA-8)
Kazakhstan, Azerbaijan, Uzbekistan, Georgia, Armenia, Tajikistan, Kyrgyzstan, Turkmenistan.

## 📂 Структура
- `data/raw/`: Исходные файлы от HPE.
- `data/processed/`: Готовые отчеты для NotebookLM.
- `scripts/`: Логика обработки.
