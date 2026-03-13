# env-edit

En enkel GUI-editor för environment variables, byggd i **Go** med **Fyne**.

## Funktioner

- Visa alla miljövariabler från aktuell process.
- Sök på både nyckel och värde.
- Skapa, redigera (inkl. rename av nyckel) och ta bort variabler.
- Läs om variabler från nuvarande process.
- Importera variabler från `.env`-fil.
- Exportera variabler till `.env`-fil.

## Kom igång

> Projektet kräver Go 1.22+.

```bash
go mod tidy
go run .
```

## Bygga för Windows

Skapa en Windows-binär (`.exe`) från valfri plattform:

```bash
GOOS=windows GOARCH=amd64 go build -o env-edit.exe .
```

(Valfritt) För ARM64 Windows:

```bash
GOOS=windows GOARCH=arm64 go build -o env-edit-arm64.exe .
```

## Notering

Appen redigerar värden i programmets minne och kan importera/exportera `.env`-filer.
Att permanent sätta systemvariabler globalt i operativsystemet (t.ex. via registry på Windows) ligger utanför denna version.
