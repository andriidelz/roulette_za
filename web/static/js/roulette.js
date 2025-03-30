/**
 * JavaScript для анимации рулетки и связанных функций
 */

document.addEventListener('DOMContentLoaded', function() {
    // Инициализация анимации рулетки
    initRouletteAnimation();
    
    // Инициализация навигации по секциям на главной странице
    initHomeSections();
    
    // Инициализация счетчиков (анимация чисел)
    initCounters();
});

/**
 * Инициализация анимации рулетки
 */
function initRouletteAnimation() {
    const rouletteWheel = document.querySelector('.roulette-wheel');
    if (!rouletteWheel) return;
    
    let rotation = 0;
    let spinning = false;
    let animationId;
    
    /**
     * Функция вращения колеса рулетки
     */
    function spinWheel() {
        if (spinning) return;
        
        spinning = true;
        let speed = 10;
        const maxSpeed = 30;
        const spinTime = 3000; // Время вращения в мс
        let startTime = null;
        
        function animate(timestamp) {
            if (!startTime) startTime = timestamp;
            const elapsedTime = timestamp - startTime;
            
            // Увеличиваем скорость в начале, затем замедляем
            if (elapsedTime < spinTime * 0.3) {
                speed = Math.min(maxSpeed, speed + 0.5);
            } else if (elapsedTime > spinTime * 0.5) {
                speed = Math.max(0.5, speed - 0.1);
            }
            
            rotation = (rotation + speed) % 360;
            rouletteWheel.style.transform = `rotate(${rotation}deg)`;
            
            if (elapsedTime < spinTime) {
                animationId = requestAnimationFrame(animate);
            } else {
                spinning = false;
            }
        }
        
        animationId = requestAnimationFrame(animate);
    }
    
    // Запускаем вращение при загрузке страницы и при клике на рулетку
    setTimeout(spinWheel, 1000);
    
    rouletteWheel.addEventListener('click', spinWheel);
}

/**
 * Инициализация навигации по секциям на главной странице
 */
function initHomeSections() {
    // Добавление класса "active" к элементам навигации при прокрутке
    window.addEventListener('scroll', function() {
        const sections = document.querySelectorAll('section[id]');
        if (!sections.length) return;
        
        const scrollPosition = window.scrollY + 100; // Небольшой отступ
        
        sections.forEach(section => {
            const sectionTop = section.offsetTop;
            const sectionHeight = section.offsetHeight;
            const sectionId = section.getAttribute('id');
            
            if (scrollPosition >= sectionTop && scrollPosition < sectionTop + sectionHeight) {
                document.querySelectorAll('.home-nav a').forEach(link => {
                    link.classList.remove('active');
                });
                
                const activeLink = document.querySelector(`.home-nav a[href="#${sectionId}"]`);
                if (activeLink) activeLink.classList.add('active');
            }
        });
    });
}

/**
 * Инициализация анимированных счетчиков
 */
function initCounters() {
    const counters = document.querySelectorAll('.stats-value');
    if (!counters.length) return;
    
    // Опции наблюдателя (когда элемент виден на 10%)
    const options = {
        root: null,
        rootMargin: '0px',
        threshold: 0.1
    };
    
    // Колбэк для наблюдателя
    const callback = (entries, observer) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                const counter = entry.target;
                const target = parseInt(counter.getAttribute('data-target'));
                
                // Проверяем, была ли уже запущена анимация
                if (!counter.classList.contains('counted')) {
                    animateCounter(counter, target);
                    counter.classList.add('counted');
                }
                
                // Прекращаем наблюдение после запуска анимации
                observer.unobserve(counter);
            }
        });
    };
    
    // Создаем наблюдатель
    const observer = new IntersectionObserver(callback, options);
    
    // Начинаем наблюдение за всеми счетчиками
    counters.forEach(counter => {
        observer.observe(counter);
    });
}

/**
 * Анимировать счетчик от 0 до целевого значения
 * @param {HTMLElement} element - Элемент счетчика
 * @param {number} target - Целевое значение
 */
function animateCounter(element, target) {
    let start = 0;
    const duration = 1500; // Длительность анимации в миллисекундах
    const increment = target / (duration / 16); // 60 кадров в секунду
    
    const updateCounter = () => {
        start += increment;
        if (start >= target) {
            element.textContent = target.toLocaleString();
            return;
        }
        
        element.textContent = Math.floor(start).toLocaleString();
        requestAnimationFrame(updateCounter);
    };
    
    updateCounter();
}
