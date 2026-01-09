import FormRecipient from "./generic/FormRecipient.js";

export default {
    name: 'Send',
    components: {
        FormRecipient
    },
    data() {
        return {
            phone: '',
            type: window.TYPEUSER,
            loading: false,
            selectedFileName: null,
            is_forwarded: false,
            audio_url: null,
            duration: 0,
            ptt: false,
        }
    },
    computed: {
        phone_id() {
            return this.phone + this.type;
        },
        isSubmitButtonDisabled() {
            if (this.type !== window.TYPEUSER && !this.phone.trim()) {
                return true;
            }

            if (!this.selectedFileName && !this.audio_url) {
                return true;
            }
            
            return false;
        }
    },
    methods: {
        openModal() {
            $('#modalAudioSend').modal({
                onApprove: function () {
                    return false;
                }
            }).modal('show');
        },
        isValidForm() {
            if (this.type !== window.TYPEUSER && !this.phone.trim()) {
                return false;
            }

            if (!this.selectedFileName && !this.audio_url) {
                return false;
            }

            // Validate duration
            if (this.duration !== 0 && (this.duration < 86400 || this.duration > 7776000)) {
                showErrorInfo("Duração inválida. Use 0 para sem expiração, ou entre 24 horas (86400s) e 90 dias (7776000s).");
                return false;
            }

            return true;
        },
        async handleSubmit() {
            if (!this.isValidForm() || this.loading) {
                return;
            }

            try {
                let response = await this.submitApi()
                showSuccessInfo(response)
                $('#modalAudioSend').modal('hide');
            } catch (err) {
                showErrorInfo(err)
            }
        },
        async submitApi() {
            this.loading = true;
            try {
                let payload = new FormData();
                payload.append("phone", this.phone_id)
                payload.append("is_forwarded", this.is_forwarded)
                payload.append("ptt", this.ptt)
                if (this.duration && this.duration > 0) {
                    payload.append("duration", this.duration)
                }

                const fileInput = $("#file_audio");
                if (fileInput.length > 0 && fileInput[0].files.length > 0) {
                    const file = fileInput[0].files[0];
                    payload.append('audio', file);
                }
                if (this.audio_url) {
                    payload.append('audio_url', this.audio_url)
                }

                const response = await window.http.post(`/send/audio`, payload)
                this.handleReset();
                return response.data.message;
            } catch (error) {
                if (error.response) {
                    throw new Error(error.response.data.message);
                }
                throw new Error(error.message);
            } finally {
                this.loading = false;
            }
        },
        handleReset() {
            this.phone = '';
            this.type = window.TYPEUSER;
            this.is_forwarded = false;
            this.duration = 0;
            this.ptt = false;
            $("#file_audio").val('');
            this.selectedFileName = null;
            this.audio_url = null;
        },
        handleFileChange(event) {
            const file = event.target.files[0];
            if (file) {
                this.selectedFileName = file.name;
            }
        }
    },
    template: `
    <div class="blue card" @click="openModal()" style="cursor: pointer">
        <div class="content">
            <a class="ui blue right ribbon label">Enviar</a>
            <div class="header">Enviar Áudio</div>
            <div class="description">
                Enviar áudio para usuário ou grupo
            </div>
        </div>
    </div>
    
    <!--  Modal SendAudio  -->
    <div class="ui small modal" id="modalAudioSend">
        <i class="close icon"></i>
        <div class="header">
            Enviar Áudio
        </div>
        <div class="content">
            <form class="ui form">
                <FormRecipient v-model:type="type" v-model:phone="phone"/>
                <div class="field">
                    <label>É Encaminhada</label>
                    <div class="ui toggle checkbox">
                        <input type="checkbox" aria-label="is forwarded" v-model="is_forwarded">
                        <label>Marcar áudio como encaminhado</label>
                    </div>
                </div>
                <div class="field">
                    <label>Nota de Voz (PTT)</label>
                    <div class="ui toggle checkbox">
                        <input type="checkbox" aria-label="ptt" v-model="ptt">
                        <label>Enviar como nota de voz (necessário para arquivos OGG/Opus)</label>
                    </div>
                </div>
                <div class="field">
                    <label>Duração de Mensagem Temporária (segundos)</label>
                    <input v-model.number="duration" type="number" min="0" max="7776000" placeholder="0 (sem expiração), 24h a 90d" aria-label="duration"/>
                </div>
                <div class="field">
                    <label>URL do Áudio</label>
                    <input type="text" v-model="audio_url" placeholder="https://example.com/audio.mp3"
                           aria-label="audio_url"/>
                </div>
                <div style="text-align: left; font-weight: bold; margin: 10px 0;">ou você pode carregar áudio do seu dispositivo
                <div class="field" style="padding-bottom: 30px">
                    <label>Áudio</label>
                    <input type="file" style="display: none" accept="audio/*" id="file_audio"
                           @change="handleFileChange"/>
                    <label for="file_audio" class="ui positive medium green left floated button" style="color: white">
                        <i class="ui upload icon"></i>
                        Carregar
                    </label>
                    <div v-if="selectedFileName" style="margin-top: 60px">
                        <div class="ui message">
                            <i class="file icon"></i>
                            Arquivo selecionado: {{ selectedFileName }}
                        </div>
                    </div>
                </div>
            </form>
        </div>
        <div class="actions">
            <button class="ui approve positive right labeled icon button" :class="{'loading': this.loading, 'disabled': isSubmitButtonDisabled || loading}"
                 @click.prevent="handleSubmit">
                Enviar
                <i class="send icon"></i>
            </button>
        </div>
    </div>
    `
}