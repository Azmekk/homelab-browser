(() => {
    const state = {
        services: [],
        editing: { id: null, icon_path: '' },
        sortable: null,
    };

    const els = {};

    document.addEventListener('DOMContentLoaded', init);

    async function init() {
        els.titleForm = document.getElementById('title-form');
        els.titleInput = document.getElementById('title-input');
        els.titleSaveBtn = document.getElementById('title-save-btn');
        els.titleSaved = document.getElementById('title-saved');
        els.addBtn = document.getElementById('add-svc-btn');
        els.emptyHint = document.getElementById('empty-hint');
        els.list = document.getElementById('svc-list');
        els.rowTpl = document.getElementById('svc-row-tpl');
        els.scrim = document.getElementById('modal-scrim');
        els.modalTitle = document.getElementById('modal-title');
        els.svcForm = document.getElementById('svc-form');
        els.svcTitleInput = document.getElementById('svc-title-input');
        els.svcUrlInput = document.getElementById('svc-url-input');
        els.svcNewtabInput = document.getElementById('svc-newtab-input');
        els.iconFile = document.getElementById('svc-icon-file');
        els.currentIcon = document.getElementById('svc-current-icon');
        els.currentIconImg = document.getElementById('svc-current-icon-img');
        els.modalError = document.getElementById('modal-error');
        els.modalSaveBtn = document.getElementById('modal-save-btn');
        els.modalCloseBtn = document.getElementById('modal-close-btn');
        els.modalCancelBtn = document.getElementById('modal-cancel-btn');
        els.brand = document.querySelector('.brand');

        els.titleInput.value = els.brand?.textContent.trim() || '';

        els.titleForm.addEventListener('submit', (e) => { e.preventDefault(); saveTitle(); });
        els.addBtn.addEventListener('click', openCreate);
        els.modalCloseBtn.addEventListener('click', closeModal);
        els.modalCancelBtn.addEventListener('click', closeModal);
        els.scrim.addEventListener('click', (e) => { if (e.target === els.scrim) closeModal(); });
        els.svcForm.addEventListener('submit', (e) => { e.preventDefault(); saveSvc(); });

        await loadServices();
    }

    async function loadServices() {
        try {
            const res = await fetch('/admin/api/services');
            if (res.status === 401) { window.location.href = '/login'; return; }
            state.services = await res.json();
            renderServices();
        } catch (e) {
            console.error(e);
        }
    }

    function renderServices() {
        els.list.innerHTML = '';
        els.emptyHint.hidden = state.services.length > 0;

        state.services.forEach((svc, idx) => {
            const row = els.rowTpl.content.firstElementChild.cloneNode(true);
            row.dataset.id = String(svc.id);

            const img = row.querySelector('.svc-icon-img');
            const placeholder = row.querySelector('.svc-icon-placeholder');
            if (svc.icon_path) {
                img.src = '/' + svc.icon_path;
                img.hidden = false;
                placeholder.hidden = true;
            } else {
                placeholder.textContent = (svc.title || '?').slice(0, 1).toUpperCase();
                placeholder.hidden = false;
                img.hidden = true;
            }

            row.querySelector('.svc-title').textContent = svc.title;
            row.querySelector('.svc-url').textContent = svc.url;

            const newtabLabel = row.querySelector('.svc-newtab');
            newtabLabel.title = svc.open_new_tab ? 'Opens in new tab' : 'Opens in same tab';
            const newtabInput = row.querySelector('.svc-newtab-input');
            newtabInput.checked = svc.open_new_tab;
            newtabInput.addEventListener('change', () => toggleNewTab(svc, newtabInput));

            const upBtn = row.querySelector('.btn-up');
            const downBtn = row.querySelector('.btn-down');
            upBtn.disabled = idx === 0;
            downBtn.disabled = idx === state.services.length - 1;
            upBtn.addEventListener('click', () => moveUp(idx));
            downBtn.addEventListener('click', () => moveDown(idx));
            row.querySelector('.btn-edit').addEventListener('click', () => openEdit(svc));
            row.querySelector('.btn-delete').addEventListener('click', () => deleteSvc(svc));

            els.list.appendChild(row);
        });

        initSortable();
    }

    function initSortable() {
        if (state.sortable) {
            state.sortable.destroy();
            state.sortable = null;
        }
        if (typeof Sortable === 'undefined') return;
        state.sortable = Sortable.create(els.list, {
            handle: '.drag-handle',
            animation: 180,
            ghostClass: 'sortable-ghost',
            onEnd: () => {
                const ids = Array.from(els.list.querySelectorAll('[data-id]')).map((el) => Number(el.dataset.id));
                const map = new Map(state.services.map((s) => [s.id, s]));
                state.services = ids.map((id) => map.get(id)).filter(Boolean);
                persistOrder();
            },
        });
    }

    async function persistOrder() {
        const order = state.services.map((s) => s.id);
        try {
            await fetch('/admin/api/services/reorder', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ order }),
            });
        } catch (e) {
            console.error(e);
        }
    }

    function moveUp(idx) {
        if (idx <= 0) return;
        const arr = state.services;
        [arr[idx - 1], arr[idx]] = [arr[idx], arr[idx - 1]];
        renderServices();
        persistOrder();
    }

    function moveDown(idx) {
        const arr = state.services;
        if (idx >= arr.length - 1) return;
        [arr[idx + 1], arr[idx]] = [arr[idx], arr[idx + 1]];
        renderServices();
        persistOrder();
    }

    async function toggleNewTab(svc, input) {
        const checked = input.checked;
        const payload = { title: svc.title, url: svc.url, open_new_tab: checked };
        try {
            const res = await fetch(`/admin/api/services/${svc.id}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
            });
            if (!res.ok) throw new Error('update failed');
            const updated = await res.json();
            svc.open_new_tab = updated.open_new_tab;
            input.closest('.svc-newtab').title = svc.open_new_tab ? 'Opens in new tab' : 'Opens in same tab';
        } catch (e) {
            console.error(e);
            input.checked = !checked;
        }
    }

    function openCreate() {
        state.editing = { id: null, icon_path: '' };
        els.modalTitle.textContent = 'Add service';
        els.svcTitleInput.value = '';
        els.svcUrlInput.value = '';
        els.svcNewtabInput.checked = false;
        els.iconFile.value = '';
        els.currentIcon.hidden = true;
        clearModalError();
        els.scrim.hidden = false;
        els.svcTitleInput.focus();
    }

    function openEdit(svc) {
        state.editing = { id: svc.id, icon_path: svc.icon_path || '' };
        els.modalTitle.textContent = 'Edit service';
        els.svcTitleInput.value = svc.title;
        els.svcUrlInput.value = svc.url;
        els.svcNewtabInput.checked = !!svc.open_new_tab;
        els.iconFile.value = '';
        if (svc.icon_path) {
            els.currentIconImg.src = '/' + svc.icon_path;
            els.currentIcon.hidden = false;
        } else {
            els.currentIcon.hidden = true;
        }
        clearModalError();
        els.scrim.hidden = false;
        els.svcTitleInput.focus();
    }

    function closeModal() {
        els.scrim.hidden = true;
        clearModalError();
    }

    function clearModalError() {
        els.modalError.textContent = '';
        els.modalError.hidden = true;
    }

    function showModalError(msg) {
        els.modalError.textContent = msg || 'Save failed';
        els.modalError.hidden = false;
    }

    async function saveSvc() {
        els.modalSaveBtn.disabled = true;
        els.modalSaveBtn.textContent = 'Saving…';
        clearModalError();
        try {
            const payload = {
                title: els.svcTitleInput.value,
                url: els.svcUrlInput.value,
                open_new_tab: !!els.svcNewtabInput.checked,
            };

            let saved;
            if (state.editing.id) {
                const res = await fetch(`/admin/api/services/${state.editing.id}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload),
                });
                if (!res.ok) throw new Error((await res.json()).error || 'update failed');
                saved = await res.json();
                const idx = state.services.findIndex((s) => s.id === saved.id);
                if (idx !== -1) state.services[idx] = saved;
            } else {
                const res = await fetch('/admin/api/services', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload),
                });
                if (!res.ok) throw new Error((await res.json()).error || 'create failed');
                saved = await res.json();
                state.services.push(saved);
            }

            if (els.iconFile.files && els.iconFile.files[0]) {
                const fd = new FormData();
                fd.append('icon', els.iconFile.files[0]);
                const up = await fetch(`/admin/api/services/${saved.id}/icon`, { method: 'POST', body: fd });
                if (!up.ok) throw new Error((await up.json()).error || 'icon upload failed');
                const withIcon = await up.json();
                const idx = state.services.findIndex((s) => s.id === withIcon.id);
                if (idx !== -1) state.services[idx] = withIcon;
            }

            renderServices();
            closeModal();
        } catch (e) {
            showModalError(e.message);
        } finally {
            els.modalSaveBtn.disabled = false;
            els.modalSaveBtn.textContent = 'Save';
        }
    }

    async function deleteSvc(svc) {
        if (!confirm(`Delete "${svc.title}"?`)) return;
        try {
            const res = await fetch(`/admin/api/services/${svc.id}`, { method: 'DELETE' });
            if (!res.ok) throw new Error('delete failed');
            state.services = state.services.filter((s) => s.id !== svc.id);
            renderServices();
        } catch (e) {
            console.error(e);
        }
    }

    async function saveTitle() {
        els.titleSaveBtn.disabled = true;
        els.titleSaved.classList.remove('visible');
        try {
            const res = await fetch('/admin/api/settings', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ page_title: els.titleInput.value }),
            });
            if (!res.ok) throw new Error('save title failed');
            const data = await res.json();
            els.titleInput.value = data.page_title;
            if (els.brand) els.brand.textContent = data.page_title;
            document.title = data.page_title;
            els.titleSaved.classList.add('visible');
            setTimeout(() => els.titleSaved.classList.remove('visible'), 1500);
        } catch (e) {
            console.error(e);
        } finally {
            els.titleSaveBtn.disabled = false;
        }
    }
})();
