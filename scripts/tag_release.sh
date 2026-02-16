#!/bin/bash

set -e

# 1. Получаем текущую дату и номер недели
YEAR=$(date +%Y)
WEEK=$(date +%V)

PREFIX="v1.${YEAR}.${WEEK}"
TAG="$PREFIX"

# 2. Ищем свободный тег (если основной занят — добавляем .1, .2, ...)
i=0
while git rev-parse "$TAG" >/dev/null 2>&1; do
  i=$((i + 1))
  TAG="${PREFIX}.${i}"
done

echo "🔧 Suggested tag: $TAG"
read -p "Edit tag or press Enter to accept [$TAG]: " CUSTOM_TAG
TAG=${CUSTOM_TAG:-$TAG}

# 3. Проверка — не занят ли вручную введённый тег
if git rev-parse "$TAG" >/dev/null 2>&1; then
  echo "❌ Tag '$TAG' already exists!"
  exit 1
fi

# 4. Предзаполненный комментарий к тегу
DEFAULT_MSG="Release version ${TAG}."
read -p "Edit tag message or press Enter to accept [$DEFAULT_MSG]: " CUSTOM_MSG
TAG_MSG=${CUSTOM_MSG:-$DEFAULT_MSG}

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
