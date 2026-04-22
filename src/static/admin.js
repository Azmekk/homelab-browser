function adminApp() {
    return {
        services: [],
        editing: { id: null, title: '', url: '', open_new_tab: false, icon_path: '' },
        modalOpen: false,
        modalError: '',
        saving: false,
        pageTitle: '',
        savingTitle: false,
        titleSaved: false,
        _sortable: null,

        async init() {
            this.pageTitle = document.querySelector('.brand')?.textContent?.trim() || '';
            await this.loadServices();
            this.$nextTick(() => this.initSortable());
        },

        initSortable() {
            if (this._sortable) {
                this._sortable.destroy();
                this._sortable = null;
            }
            const list = this.$refs.list;
            if (!list || typeof Sortable === 'undefined') return;
            this._sortable = Sortable.create(list, {
                handle: '.drag-handle',
                animation: 180,
                ghostClass: 'sortable-ghost',
                onEnd: () => {
                    const ids = Array.from(list.querySelectorAll('[data-id]'))
                        .map((el) => Number(el.dataset.id));
                    // Reorder local array to match DOM, then persist.
                    const map = new Map(this.services.map((s) => [s.id, s]));
                    this.services = ids.map((id) => map.get(id)).filter(Boolean);
                    this.persistOrder();
                },
            });
        },

        async loadServices() {
            try {
                const res = await fetch('/admin/api/services');
                if (res.status === 401) { window.location.href = '/login'; return; }
                this.services = await res.json();
            } catch (e) {
                console.error(e);
            }
        },

        async persistOrder() {
            const order = this.services.map((s) => s.id);
            try {
                await fetch('/admin/api/services/reorder', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ order }),
                });
            } catch (e) {
                console.error(e);
            }
        },

        moveUp(idx) {
            if (idx <= 0) return;
            const copy = this.services.slice();
            [copy[idx - 1], copy[idx]] = [copy[idx], copy[idx - 1]];
            this.services = copy;
            this.persistOrder();
        },

        moveDown(idx) {
            if (idx >= this.services.length - 1) return;
            const copy = this.services.slice();
            [copy[idx + 1], copy[idx]] = [copy[idx], copy[idx + 1]];
            this.services = copy;
            this.persistOrder();
        },

        async toggleNewTab(svc, checked) {
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
            } catch (e) {
                console.error(e);
                svc.open_new_tab = !checked;
            }
        },

        openCreate() {
            this.editing = { id: null, title: '', url: '', open_new_tab: false, icon_path: '' };
            this.modalError = '';
            this.modalOpen = true;
        },

        openEdit(svc) {
            this.editing = { ...svc };
            this.modalError = '';
            this.modalOpen = true;
        },

        closeModal() {
            this.modalOpen = false;
            this.modalError = '';
        },

        async saveSvc() {
            this.saving = true;
            this.modalError = '';
            try {
                const payload = {
                    title: this.editing.title,
                    url: this.editing.url,
                    open_new_tab: !!this.editing.open_new_tab,
                };

                let saved;
                if (this.editing.id) {
                    const res = await fetch(`/admin/api/services/${this.editing.id}`, {
                        method: 'PUT',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify(payload),
                    });
                    if (!res.ok) throw new Error((await res.json()).error || 'update failed');
                    saved = await res.json();
                    const idx = this.services.findIndex((s) => s.id === saved.id);
                    if (idx !== -1) this.services[idx] = saved;
                } else {
                    const res = await fetch('/admin/api/services', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify(payload),
                    });
                    if (!res.ok) throw new Error((await res.json()).error || 'create failed');
                    saved = await res.json();
                    this.services.push(saved);
                }

                const fileInput = this.$refs.iconFile;
                if (fileInput && fileInput.files && fileInput.files[0]) {
                    const fd = new FormData();
                    fd.append('icon', fileInput.files[0]);
                    const up = await fetch(`/admin/api/services/${saved.id}/icon`, { method: 'POST', body: fd });
                    if (!up.ok) throw new Error((await up.json()).error || 'icon upload failed');
                    const withIcon = await up.json();
                    const idx = this.services.findIndex((s) => s.id === withIcon.id);
                    if (idx !== -1) this.services[idx] = withIcon;
                }

                this.modalOpen = false;
            } catch (e) {
                this.modalError = e.message || 'Save failed';
            } finally {
                this.saving = false;
            }
        },

        async deleteSvc(svc) {
            if (!confirm(`Delete "${svc.title}"?`)) return;
            try {
                const res = await fetch(`/admin/api/services/${svc.id}`, { method: 'DELETE' });
                if (!res.ok) throw new Error('delete failed');
                this.services = this.services.filter((s) => s.id !== svc.id);
            } catch (e) {
                console.error(e);
            }
        },

        async saveTitle() {
            this.savingTitle = true;
            this.titleSaved = false;
            try {
                const res = await fetch('/admin/api/settings', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ page_title: this.pageTitle }),
                });
                if (!res.ok) throw new Error('save title failed');
                const data = await res.json();
                this.pageTitle = data.page_title;
                const brand = document.querySelector('.brand');
                if (brand) brand.textContent = this.pageTitle;
                document.title = this.pageTitle;
                this.titleSaved = true;
                setTimeout(() => { this.titleSaved = false; }, 1500);
            } catch (e) {
                console.error(e);
            } finally {
                this.savingTitle = false;
            }
        },
    };
}
