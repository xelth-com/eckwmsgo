@echo off
echo 🧹 Cleaning up Windows ghosts...

REM 1. Удаляем файл nul, если он существует (используем специальный префикс для обхода защиты Windows)
if exist nul (
    del /f /q \\.\%CD%\nul 2>nul
    if %errorlevel% equ 0 (
        echo ✅ Deleted 'nul' file
    ) else (
        echo ⚠️ Could not delete 'nul' file
    )
) else (
    echo ℹ️ 'nul' file not found ^(good^)
)

REM 2. Удаляем старые файлы из индекса Git (если они физически удалены, но висят в git status)
echo.
echo 🔍 Synchronizing Git index...
git rm --cached internal/models/item.go 2>nul
git rm --cached internal/models/warehouse.go 2>nul
git rm --cached internal/models/odoo_structs.go 2>nul

REM 3. Добавляем все изменения
git add .
git add .eck/AnswerToSA.md

echo.
echo ✅ Git index synchronized.
echo.
echo 📊 Current status:
git status --short | findstr /v "tmpclaude"
