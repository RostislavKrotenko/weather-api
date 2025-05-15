async function fetchWeather() {
    const city = document.getElementById('weather-city').value;
    const resEl = document.getElementById('weather-result');
    resEl.textContent = '';
    if (!city) { resEl.textContent = 'Please enter a city.'; return; }
    try {
      const r = await fetch(`/api/weather?city=${encodeURIComponent(city)}`);
      if (!r.ok) throw new Error(r.statusText);
      const j = await r.json();
      resEl.textContent =
        `Temp: ${j.temperature}°C, Humidity: ${j.humidity}%, ${j.description}`;
    } catch (e) {
      resEl.textContent = 'Error: ' + e.message;
    }
  }
  
  async function subscribe() {
    const email = document.getElementById('sub-email').value;
    const city = document.getElementById('sub-city').value;
    const frequency = document.getElementById('sub-frequency').value;
    const msg = document.getElementById('sub-msg');
    msg.textContent = '';
    try {
      const r = await fetch('/api/subscribe', {
        method: 'POST',
        headers: {'Content-Type':'application/json'},
        body: JSON.stringify({ email, city, frequency })
      });
      const j = await r.json();
      if (r.ok) {
        msg.style.color = 'green';
        msg.textContent = `Subscribed! Token: ${j.token}`;
      } else {
        msg.style.color = 'red';
        msg.textContent = j.message || 'Error';
      }
    } catch (e) {
      msg.style.color = 'red';
      msg.textContent = 'Network error';
    }
  }
  
  async function confirmSub() {
    const token = document.getElementById('token').value;
    const msg = document.getElementById('token-msg');
    msg.textContent = '';
    if (!token) { msg.textContent = 'Token required.'; return; }
    try {
      const r = await fetch(`/api/confirm/${encodeURIComponent(token)}`);
      if (r.ok) {
        msg.style.color = 'green';
        msg.textContent = 'Subscription confirmed!';
      } else {
        const j = await r.json().catch(()=>({message:r.statusText}));
        msg.style.color = 'red';
        msg.textContent = j.message || 'Error';
      }
    } catch (e) {
      msg.style.color = 'red';
      msg.textContent = 'Network error';
    }
  }
  
  async function unsubscribe() {
    const token = document.getElementById('token').value;
    const msg = document.getElementById('token-msg');
    msg.textContent = '';
    if (!token) { msg.textContent = 'Token required.'; return; }
    try {
      const r = await fetch(`/api/unsubscribe/${encodeURIComponent(token)}`);
      if (r.ok) {
        msg.style.color = 'green';
        msg.textContent = 'Unsubscribed successfully.';
      } else {
        const j = await r.json().catch(()=>({message:r.statusText}));
        msg.style.color = 'red';
        msg.textContent = j.message || 'Error';
      }
    } catch (e) {
      msg.style.color = 'red';
      msg.textContent = 'Network error';
    }
  }