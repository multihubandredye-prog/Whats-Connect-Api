export default {
    name: 'DeviceManager',
    props: {
        wsBasePath: {
            type: String,
            default: ''
        }
    },
    data() {
        return {
            deviceList: [],
            selectedDeviceId: '',
            deviceIdInput: '',
            isCreatingDevice: false,
            deviceToDelete: { id: '', jid: '', state: '' },
            isDeleting: false
        }
    },
    computed: {
        selectedDevice() {
            if (!this.selectedDeviceId) return null;
            return this.deviceList.find(d => (d.id || d.device) === this.selectedDeviceId) || null;
        },
        isSelectedDeviceLoggedIn() {
            return this.selectedDevice?.state === 'logged_in';
        }
    },
    methods: {
        async fetchDevices() {
            try {
                const res = await window.http.get(`/devices`);
                this.deviceList = res.data.results || [];
                if (!this.selectedDeviceId && this.deviceList.length > 0) {
                    const first = this.deviceList[0].id || this.deviceList[0].device;
                    this.setDeviceContext(first);
                }
                // Emit devices to parent for other components
                this.$emit('devices-updated', this.deviceList);
            } catch (err) {
                console.error(err);
            }
        },
        setDeviceContext(id) {
            if (!id) {
                showErrorInfo('ID do Dispositivo é obrigatório');
                return;
            }
            this.selectedDeviceId = id;
            this.$emit('device-selected', id);
            showSuccessInfo(`Usando o dispositivo ${id}`);
        },
        async createDevice() {
            try {
                this.isCreatingDevice = true;
                const payload = this.deviceIdInput ? {device_id: this.deviceIdInput} : {};
                const res = await window.http.post('/devices', payload);
                const deviceID = res.data?.results?.id || res.data?.results?.device_id || this.deviceIdInput;
                this.setDeviceContext(deviceID);
                this.deviceIdInput = '';
            } catch (err) {
                const msg = err.response?.data?.message || err.message || 'Falha ao criar dispositivo';
                showErrorInfo(msg);
            } finally {
                this.isCreatingDevice = false;
            }
        },
        useDeviceFromInput() {
            if (!this.deviceIdInput) {
                showErrorInfo('Insira um device_id ou crie um primeiro.');
                return;
            }
            this.setDeviceContext(this.deviceIdInput);
        },
        openDeleteModal(deviceId, jid) {
            const device = this.deviceList.find(d => (d.id || d.device) === deviceId);
            this.deviceToDelete = { id: deviceId, jid: jid || '', state: device?.state || '' };
            $('#deleteDeviceModal').modal({
                closable: false,
                onApprove: () => {
                    this.executeDelete();
                    return false;
                },
                onDeny: () => {
                    this.resetDeleteState();
                }
            }).modal('show');
        },
        resetDeleteState() {
            this.deviceToDelete = { id: '', jid: '', state: '' };
            this.isDeleting = false;
        },
        async executeDelete() {
            const deviceId = this.deviceToDelete.id;
            if (!deviceId) {
                showErrorInfo('Nenhum dispositivo selecionado para exclusão');
                return;
            }
            try {
                this.isDeleting = true;
                
                // Logout first (fire and forget), then delete
                window.http.get(`/app/logout`, {
                    headers: { 'X-Device-Id': encodeURIComponent(deviceId) }
                }).catch(() => {});
                
                await window.http.delete(`/devices/${encodeURIComponent(deviceId)}`);
                showSuccessInfo(`Dispositivo ${deviceId} excluído com sucesso`);
                $('#deleteDeviceModal').modal('hide');
                
                if (this.selectedDeviceId === deviceId) {
                    this.selectedDeviceId = '';
                    this.$emit('device-selected', '');
                }
                
                await this.fetchDevices();
                this.resetDeleteState();
            } catch (err) {
                const msg = err.response?.data?.message || err.message || 'Falha ao excluir dispositivo';
                showErrorInfo(msg);
                this.isDeleting = false;
            }
        },
        // Called by parent to refresh devices
        refresh() {
            this.fetchDevices();
        },
        // Called by parent to update device list from websocket
        updateDeviceList(devices) {
            if (Array.isArray(devices)) {
                this.deviceList = devices;
                this.$emit('devices-updated', devices);
            }
        }
    },
    mounted() {
        this.fetchDevices();
    },
    template: `
    <div class="ui stackable grid">
        <div class="ten wide column">
            <div class="ui segment">
                <h3 class="ui header">
                    <i class="play icon"></i>
                    <div class="content">
                        Configuração do Dispositivo
                        <div class="sub header">Crie ou selecione um device_id e, em seguida, abra o login.</div>
                    </div>
                </h3>
                <div class="ui form">
                    <div class="two fields">
                        <div class="field">
                            <label>ID do Dispositivo (opcional)</label>
                            <input type="text" v-model="deviceIdInput" placeholder="Deixe vazio para gerar automaticamente">
                        </div>
                        <div class="field">
                            <label>Ações</label>
                            <div class="ui buttons">
                                <button class="ui primary button" :class="{loading: isCreatingDevice}" @click="createDevice">
                                    Criar dispositivo
                                </button>
                                <div class="or"></div>
                                <button class="ui button" @click="useDeviceFromInput">Usar este dispositivo</button>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="ui divider"></div>
                
                <!-- Device List -->
                <div class="ui relaxed list" v-if="deviceList.length">
                    <div class="item" v-for="dev in deviceList" :key="dev.id || dev.device">
                        <i class="mobile alternate icon"></i>
                        <div class="content">
                            <div class="header">{{ dev.id || dev.device }}</div>
                            <div class="description">
                                <span>Estado: {{ dev.state || 'desconhecido' }}</span>
                                <span v-if="dev.jid"> · JID: {{ dev.jid }}</span>
                            </div>
                        </div>
                        <div class="right floated content">
                            <button class="ui mini button" 
                                    :class="{active: selectedDeviceId === (dev.id || dev.device)}"
                                    @click="setDeviceContext(dev.id || dev.device)">
                                {{ selectedDeviceId === (dev.id || dev.device) ? 'Selecionado' : 'Usar' }}
                            </button>
                            <button class="ui mini red icon button" 
                                    @click="openDeleteModal(dev.id || dev.device, dev.jid)" 
                                    :class="{loading: isDeleting && deviceToDelete.id === (dev.id || dev.device)}">
                                <i class="trash icon" style="margin: 0;"></i>
                            </button>
                        </div>
                    </div>
                </div>
                <div class="ui message" v-else>
                    Nenhum dispositivo ainda. Crie um para começar.
                </div>
            </div>
        </div>
        <div class="six wide column">
            <div class="ui warning message">
                <div class="header">Como fazer login</div>
                <ul class="list">
                    <li>Passo 1: Crie um dispositivo para obter o <code>device_id</code>.</li>
                    <li>Passo 2: Envie <code>X-Device-Id: device_id</code> nas chamadas REST.</li>
                    <li>Passo 3: Abra o cartão de Login para emparelhar (QR ou código).</li>
                    <li>URL do WebSocket: <code>{{ wsBasePath }}/ws?device_id=&lt;device_id&gt;</code></li>
                </ul>
            </div>
        </div>

        <!-- Delete Device Confirmation Modal -->
        <div class="ui small modal" id="deleteDeviceModal">
            <div class="header">
                <i class="trash alternate icon"></i>
                Confirmar Exclusão do Dispositivo
            </div>
            <div class="content">
                <p>Tem certeza de que deseja excluir este dispositivo?</p>
                <div class="ui segment">
                    <p><strong>ID do Dispositivo:</strong> <code>{{ deviceToDelete.id }}</code></p>
                    <p v-if="deviceToDelete.jid"><strong>JID:</strong> <code>{{ deviceToDelete.jid }}</code></p>
                </div>
                <div class="ui warning message">
                    <div class="header">Aviso</div>
                    <p>Esta ação excluirá permanentemente o dispositivo e todos os dados associados, incluindo chats e mensagens. Esta ação não pode ser desfeita.</p>
                </div>
            </div>
            <div class="actions">
                <button class="ui cancel button">Cancelar</button>
                <button class="ui red approve button" :class="{loading: isDeleting}">
                    <i class="trash icon"></i>
                    Excluir Dispositivo
                </button>
            </div>
        </div>
    </div>
    `
}
