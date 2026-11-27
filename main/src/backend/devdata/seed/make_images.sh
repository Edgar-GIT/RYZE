#!/usr/bin/env bash
# Generates the demo images for `go run . seed-demo` (covers and feedback).
set -euo pipefail
OUT="$1"
FONT=Adwaita-Sans-Black
SUB=Adwaita-Sans-Bold
cover() { # name c1 c2 c3 title subtitle seed
  local name=$1 c1=$2 c2=$3 c3=$4 title=$5 sub=$6 seed=$7
  magick -size 1200x800 -define gradient:angle=$((120 + seed % 5 * 12)) gradient:"$c1"-"$c2" \
    \( -size 1200x800 xc:none -fill "$c3" -draw "circle $((700 + seed % 4 * 90)),$((200 + seed % 3 * 80)) $((700 + seed % 4 * 90)),$((440 + seed % 3 * 80))" -blur 0x90 -channel A -evaluate multiply 0.55 +channel \) -compose screen -composite \
    \( -size 1200x800 xc:none -fill "rgba(255,255,255,0.08)" -draw "circle 980,180 980,520" -draw "circle 160,700 160,900" \) -compose over -composite \
    \( -size 1200x800 gradient:"rgba(2,4,31,0)"-"rgba(2,4,31,0.78)" \) -compose over -composite \
    -font "$FONT" -pointsize 118 -fill white -gravity southwest -annotate +64+150 "$title" \
    -font "$SUB" -pointsize 40 -fill "#7cebff" -gravity southwest -annotate +68+86 "$sub" \
    -quality 82 "$OUT/$name.jpg"
}
# Free plans: one per category.
cover free_hypertrophy "#072a77" "#00cbff" "#2a8cff" "HYPERTROPHY" "Plano grátis · 4 semanas" 11
cover free_strength "#02041f" "#ff6b3d" "#ffb020" "STRENGTH" "Plano grátis · 4 semanas" 12
cover free_hyrox "#0b1a3a" "#ffd400" "#18d8ff" "HYROX" "Plano grátis · 4 semanas" 13
cover free_crossfit "#1a0b2e" "#ff3d7f" "#7c3aed" "CROSSFIT" "Plano grátis · 4 semanas" 14
cover free_fat_loss "#03241f" "#2fe3a8" "#00cbff" "FAT LOSS" "Plano grátis · 4 semanas" 15
cover free_at_home "#1d1430" "#ffa3c7" "#b69bff" "AT HOME" "Plano grátis · 4 semanas" 16
# Premium Level 1 packages.
cover l1_muscle_all "#072a77" "#18d8ff" "#ffffff" "MASSA TOTAL" "Premium Nível 1 · Ganhar músculo" 21
cover l1_muscle_female "#3a0b2e" "#ff7ac6" "#ffd1e8" "GLÚTEO & FORÇA" "Premium Nível 1 · Ganhar músculo" 22
cover l1_muscle_male "#0a1636" "#5aa8ff" "#2a5bff" "V-SHAPE" "Premium Nível 1 · Ganhar músculo" 23
cover l1_fat_all "#04261f" "#34d399" "#a7f3d0" "SECAR 12" "Premium Nível 1 · Perder gordura" 24
cover l1_fat_female "#2b0f36" "#e879f9" "#fbcfe8" "DEFINIR" "Premium Nível 1 · Perder gordura" 25
cover l1_fat_male "#1f1205" "#fb923c" "#fde68a" "CUT PRO" "Premium Nível 1 · Perder gordura" 26
cover l1_strength_all "#140505" "#ef4444" "#fca5a5" "FORÇA 5x5" "Premium Nível 1 · Força e condição" 27
cover l1_strength_male "#05121f" "#94a3b8" "#e2e8f0" "POWERBUILD" "Premium Nível 1 · Força e condição" 28
cover l1_sport_all "#06210d" "#22c55e" "#bbf7d0" "FUTEBOL" "Premium Nível 1 · Desporto" 29
cover l1_sport_female "#0a1f3a" "#38bdf8" "#e0f2fe" "PADEL" "Premium Nível 1 · Desporto" 30
# Feedback pictures: different scenes and moods.
fb() { # name c1 c2 big small seed
  local name=$1 c1=$2 c2=$3 big=$4 small=$5 seed=$6
  magick -size 1000x1250 -define gradient:angle=$((100 + seed % 6 * 15)) gradient:"$c1"-"$c2" \
    \( -size 1000x1250 xc:none -fill "rgba(255,255,255,0.9)" -draw "circle $((300 + seed % 5 * 90)),$((260 + seed % 4 * 120)) $((300 + seed % 5 * 90)),$((460 + seed % 4 * 120))" -blur 0x120 -channel A -evaluate multiply 0.45 +channel \) -compose screen -composite \
    \( -size 1000x1250 xc:none -fill "rgba(255,255,255,0.10)" -draw "roundrectangle 90,140 910,1110 60,60" \) -compose over -composite \
    -font "$FONT" -pointsize 150 -fill white -gravity center -annotate +0-80 "$big" \
    -font "$SUB" -pointsize 52 -fill "rgba(255,255,255,0.85)" -gravity center -annotate +0+70 "$small" \
    -quality 80 "$OUT/$name.jpg"
}
fb feedback01 "#072a77" "#00cbff" "-8 kg" "em 12 semanas" 41
fb feedback02 "#3a0b2e" "#ff7ac6" "100 kg" "primeiro supino" 42
fb feedback03 "#04261f" "#34d399" "DIA 90" "sem falhar um treino" 43
fb feedback04 "#1f1205" "#fb923c" "PR" "agachamento 140 kg" 44
fb feedback05 "#0a1f3a" "#38bdf8" "HYROX" "primeira prova feita" 45
fb feedback06 "#140505" "#ef4444" "5x5" "força a subir" 46
fb feedback07 "#2b0f36" "#e879f9" "+4 kg" "de músculo limpo" 47
fb feedback08 "#06210d" "#22c55e" "10 km" "a correr sem parar" 48
fb feedback09 "#0b1a3a" "#ffd400" "EM CASA" "só com halteres" 49
fb feedback10 "#1d1430" "#b69bff" "1 ANO" "de RYZE" 50
