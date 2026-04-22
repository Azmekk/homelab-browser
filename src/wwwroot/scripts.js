// Dashboard: live clock + weather.
(function () {
    const clockEl = document.getElementById('clock');
    if (clockEl) {
        const fmt = new Intl.DateTimeFormat(undefined, {
            hour: '2-digit',
            minute: '2-digit',
            hour12: false,
        });
        const tick = () => { clockEl.textContent = fmt.format(new Date()); };
        tick();
        setInterval(tick, 1000 * 15);
        // Short extra tick at minute boundary so we're not always behind by up to 15s.
        const msToNextMinute = 60000 - (Date.now() % 60000);
        setTimeout(tick, msToNextMinute);
    }

    const weatherEl = document.getElementById('weather');
    if (!weatherEl) return;

    const iconEl = weatherEl.querySelector('.weather-icon');
    const tempEl = weatherEl.querySelector('.weather-temp');
    const locEl = weatherEl.querySelector('.weather-loc');

    const codeToIcon = (code) => {
        if (code === 0) return '☀';
        if (code <= 3) return '⛅';
        if (code <= 48) return '☁';
        if (code <= 67) return '🌧';
        if (code <= 77) return '❄';
        if (code <= 82) return '🌧';
        if (code <= 86) return '🌨';
        if (code <= 99) return '⛈';
        return '◌';
    };

    const setWeather = (data, locationName) => {
        if (!data || !data.current) return;
        const t = Math.round(data.current.temperature_2m);
        iconEl.textContent = codeToIcon(data.current.weather_code);
        tempEl.textContent = `${t}°`;
        locEl.textContent = locationName || '';
    };

    const fetchWeather = async (lat, lon, locName) => {
        const url = `https://api.open-meteo.com/v1/forecast?latitude=${lat}&longitude=${lon}&current=temperature_2m,weather_code&timezone=auto`;
        try {
            const res = await fetch(url);
            if (!res.ok) throw new Error('weather http');
            const data = await res.json();
            setWeather(data, locName);
        } catch {
            iconEl.textContent = '◌';
            tempEl.textContent = '';
            locEl.textContent = 'Weather unavailable';
        }
    };

    const reverseGeocode = async (lat, lon) => {
        try {
            const url = `https://geocoding-api.open-meteo.com/v1/reverse?latitude=${lat}&longitude=${lon}&count=1&language=en&format=json`;
            const res = await fetch(url);
            if (!res.ok) return '';
            const data = await res.json();
            const hit = data.results && data.results[0];
            return hit ? (hit.name || '') : '';
        } catch {
            return '';
        }
    };

    const fromIP = async () => {
        try {
            const res = await fetch('https://ipapi.co/json/');
            if (!res.ok) throw new Error('ip http');
            const d = await res.json();
            if (d.latitude && d.longitude) {
                fetchWeather(d.latitude, d.longitude, d.city || d.region || '');
            }
        } catch {
            iconEl.textContent = '◌';
            tempEl.textContent = '';
            locEl.textContent = 'Weather unavailable';
        }
    };

    if (navigator.geolocation) {
        navigator.geolocation.getCurrentPosition(
            async (pos) => {
                const loc = await reverseGeocode(pos.coords.latitude, pos.coords.longitude);
                fetchWeather(pos.coords.latitude, pos.coords.longitude, loc);
            },
            () => fromIP(),
            { timeout: 4000, maximumAge: 3600_000 }
        );
    } else {
        fromIP();
    }
})();
