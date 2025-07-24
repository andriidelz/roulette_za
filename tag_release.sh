#!/bin/bash

set -e

# 1. Получаем текущую дату и номер недели
YEAR=$(date +%Y)
WEEK=$(date +%V)  # ISO неделя (01–53)

# 2. Предлагаем версию
DEFAULT_TAG="v1.${YEAR}.${WEEK}"

echo "🔧 Default tag: $DEFAULT_TAG"
read -p "Edit tag or press Enter to accept [$DEFAULT_TAG]: " CUSTOM_TAG
TAG=${CUSTOM_TAG:-$DEFAULT_TAG}

# 3. Проверка — существует ли уже такой тег
if git rev-parse "$TAG" >/dev/null 2>&1; then
  echo "❌ Tag '$TAG' already exists!"
  exit 1
fi

# 4. Комментарий к тегу
read -p "Enter tag message (annotated tag): " TAG_MSG
TAG_MSG=${TAG_MSG:-"Release $TAG"}

# 5. Подтверждение
echo ""
echo "📌 About to create and push tag:"
echo "    Tag:     $TAG"
echo "    Message: $TAG_MSG"
echo ""
read -p "Proceed? [y/N]: " CONFIRM

if [[ "$CONFIRM" != "y" && "$CONFIRM" != "Y" ]]; then
  echo "❌ Cancelled."
  exit 1
fi

# 6. Создаём аннотированный тег
git tag -a "$TAG" -m "$TAG_MSG"

# 7. Пушим тег
git push origin "$TAG"

echo "✅ Tag '$TAG' created and pushed."
