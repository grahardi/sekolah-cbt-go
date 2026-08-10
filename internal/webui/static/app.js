// Vanilla JS on purpose — no bundler, no framework, no CDN. This runs on
// school computers with unreliable internet during a live exam, so the
// fewer moving parts, the better.
//
// Session is kept in sessionStorage (survives an accidental reload within
// the same tab, cleared when the tab closes) rather than a longer-lived
// store — if it's lost, re-login is safe: the backend resumes the same
// siswa_ujian attempt and returns previously saved jawaban, so nothing is
// lost, just re-entered credentials.

const state = {
  token: null,
  expiresAt: null,
  nama: null,
  soal: [],
  currentIndex: 0,
  timerInterval: null,
};

function el(id) {
  return document.getElementById(id);
}

function showScreen(name) {
  ["login", "exam", "result"].forEach((s) => {
    el(s + "-screen").classList.toggle("hidden", s !== name);
  });
}

function pad(n) {
  return String(n).padStart(2, "0");
}

// --- session persistence -------------------------------------------------

function loadSession() {
  const raw = sessionStorage.getItem("cbt_session");
  if (!raw) return false;
  try {
    const parsed = JSON.parse(raw);
    state.token = parsed.token;
    state.expiresAt = new Date(parsed.expiresAt);
    state.nama = parsed.nama;
    return true;
  } catch {
    return false;
  }
}

function saveSession() {
  sessionStorage.setItem(
    "cbt_session",
    JSON.stringify({
      token: state.token,
      expiresAt: state.expiresAt.toISOString(),
      nama: state.nama,
    })
  );
}

function clearSession() {
  sessionStorage.removeItem("cbt_session");
  state.token = null;
}

// --- API helper ------------------------------------------------------------

async function apiFetch(path, options = {}) {
  const headers = options.headers || {};
  const hadToken = !!state.token;
  if (state.token) headers["Authorization"] = "Bearer " + state.token;
  if (options.body) headers["Content-Type"] = "application/json";

  const res = await fetch(path, { ...options, headers });

  // Only treat 401 as "session expired" when the request actually carried a
  // bearer token — /peserta/login has no token yet and legitimately
  // returns 401 for wrong credentials, which is a different message
  // entirely and shouldn't be papered over with a generic one.
  if (res.status === 401 && hadToken) {
    clearSession();
    showScreen("login");
    setLoginError("Sesi berakhir, silakan login ulang.");
    throw new Error("unauthorized");
  }

  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error || "Terjadi kesalahan, coba lagi.");
  }
  return data;
}

function setLoginError(msg) {
  el("login-error").textContent = msg || "";
}

// --- login -----------------------------------------------------------------

el("login-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  setLoginError("");

  const no_ujian = el("no_ujian").value.trim();
  const password = el("password").value;
  const token = el("token").value.trim().toUpperCase();

  const submitBtn = e.target.querySelector("button[type=submit]");
  submitBtn.disabled = true;

  try {
    const data = await apiFetch("/peserta/login", {
      method: "POST",
      body: JSON.stringify({ no_ujian, password, token }),
    });
    state.token = data.access_token;
    state.expiresAt = new Date(data.expires_at);
    state.nama = data.nama;
    saveSession();
    await startExam();
  } catch (err) {
    setLoginError(err.message);
  } finally {
    submitBtn.disabled = false;
  }
});

// --- exam --------------------------------------------------------------

async function startExam() {
  el("peserta-nama").textContent = state.nama;
  const data = await apiFetch("/ujian/soal");
  state.soal = data.soal || [];
  state.currentIndex = 0;

  if (state.soal.length === 0) {
    alert("Belum ada soal untuk ujian ini. Hubungi panitia ujian.");
    return;
  }

  renderPills();
  renderSoal();
  showScreen("exam");
  startTimer();
}

function startTimer() {
  clearInterval(state.timerInterval);
  tickTimer();
  state.timerInterval = setInterval(tickTimer, 1000);
}

function tickTimer() {
  const remainingMs = state.expiresAt - new Date();
  let remaining = Math.floor(remainingMs / 1000);

  if (remaining <= 0) {
    remaining = 0;
    clearInterval(state.timerInterval);
    finishExam(true);
  }

  const h = Math.floor(remaining / 3600);
  const m = Math.floor((remaining % 3600) / 60);
  const s = remaining % 60;
  el("timer").textContent = h > 0 ? `${pad(h)}:${pad(m)}:${pad(s)}` : `${pad(m)}:${pad(s)}`;

  const box = el("timer-box");
  box.classList.toggle("warn", remaining <= 600 && remaining > 120);
  box.classList.toggle("danger", remaining <= 120);
}

function isAnswered(s) {
  return !!(s.jawaban_soal_id || (s.jawaban_text && s.jawaban_text.trim() !== ""));
}

function renderPills() {
  const wrap = el("pills");
  wrap.innerHTML = "";
  state.soal.forEach((s, i) => {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className =
      "pill" + (isAnswered(s) ? " answered" : "") + (i === state.currentIndex ? " active" : "");
    btn.textContent = String(i + 1);
    btn.setAttribute("aria-label", `Soal ${i + 1}${isAnswered(s) ? ", sudah dijawab" : ""}`);
    btn.addEventListener("click", () => {
      state.currentIndex = i;
      renderSoal();
      renderPills();
    });
    wrap.appendChild(btn);
  });
}

function renderSoal() {
  const s = state.soal[state.currentIndex];

  el("soal-nomor").textContent = `Soal ${state.currentIndex + 1} dari ${state.soal.length}`;

  el("soal-rujukan").classList.toggle("hidden", !s.rujukan);
  el("soal-rujukan").textContent = s.rujukan || "";

  const audioEl = el("soal-audio");
  audioEl.classList.toggle("hidden", !s.audio);
  if (s.audio) audioEl.src = s.audio;

  el("soal-pertanyaan").textContent = s.pertanyaan;

  const opsiWrap = el("soal-opsi");
  opsiWrap.innerHTML = "";
  el("save-status").textContent = "";

  if (s.tipe_soal === 2) {
    const textarea = document.createElement("textarea");
    textarea.rows = 6;
    textarea.placeholder = "Tulis jawaban di sini...";
    textarea.value = s.jawaban_text || "";
    textarea.addEventListener("change", () => submitJawaban(s, { jawaban_text: textarea.value }));
    opsiWrap.appendChild(textarea);
  } else {
    (s.opsi || []).forEach((opt) => {
      const label = document.createElement("label");
      label.className = "opsi";

      const radio = document.createElement("input");
      radio.type = "radio";
      radio.name = "opsi-" + s.id;
      radio.value = opt.id;
      radio.checked = s.jawaban_soal_id === opt.id;
      radio.addEventListener("change", () => submitJawaban(s, { jawaban_soal_id: opt.id }));

      const span = document.createElement("span");
      span.textContent = opt.teks;

      label.appendChild(radio);
      label.appendChild(span);
      opsiWrap.appendChild(label);
    });
  }

  el("btn-prev").disabled = state.currentIndex === 0;
  el("btn-next").disabled = state.currentIndex === state.soal.length - 1;
}

async function submitJawaban(soal, payload) {
  el("save-status").textContent = "Menyimpan...";
  try {
    await apiFetch("/ujian/jawab", {
      method: "POST",
      body: JSON.stringify({ soal_id: soal.id, ...payload }),
    });
    Object.assign(soal, payload);
    el("save-status").textContent = "Tersimpan";
    renderPills();
  } catch (err) {
    if (err.message !== "unauthorized") {
      el("save-status").textContent = "Gagal menyimpan: " + err.message;
    }
  }
}

el("btn-prev").addEventListener("click", () => {
  state.currentIndex--;
  renderSoal();
  renderPills();
});

el("btn-next").addEventListener("click", () => {
  state.currentIndex++;
  renderSoal();
  renderPills();
});

el("btn-selesai").addEventListener("click", () => {
  el("confirm-dialog").classList.remove("hidden");
});

el("confirm-cancel").addEventListener("click", () => {
  el("confirm-dialog").classList.add("hidden");
});

el("confirm-ok").addEventListener("click", () => finishExam(false));

async function finishExam(auto) {
  el("confirm-dialog").classList.add("hidden");
  clearInterval(state.timerInterval);
  try {
    const data = await apiFetch("/ujian/selesai", { method: "POST" });
    clearSession();
    el("result-benar").textContent = data.jumlah_benar;
    el("result-salah").textContent = data.jumlah_salah;
    el("result-nilai").textContent = Number(data.nilai).toFixed(2);
    el("result-auto").classList.toggle("hidden", !auto);
    showScreen("result");
  } catch (err) {
    if (err.message !== "unauthorized") {
      alert("Gagal menyelesaikan ujian: " + err.message);
    }
  }
}

// --- bootstrap -----------------------------------------------------------

if (loadSession() && state.expiresAt > new Date()) {
  startExam().catch(() => {
    clearSession();
    showScreen("login");
  });
} else {
  clearSession();
  showScreen("login");
}
