import FormRecipient from "./generic/FormRecipient.js";

export default {
    name: 'SendFile',
    components: {
        FormRecipient
    },
    props: {
        maxFileSize: {
            type: String,
            required: true,
        }
    },
    data() {
        return {
            caption: '',
            type: window.TYPEUSER,
            phone: '',
            loading: false,
            selectedFileName: null,
            is_forwarded: false,
            duration: 0
        }
    },
    computed: {
        phone_id() {
            return this.phone + this.type;
        },
        isSubmitButtonDisabled() {
            if (this.type !== window.TYPESTATUS && !this.phone.trim()) {
                return true;
            }

            if (!this.selectedFileName) {
                return true;
            }
            
            return false;
        }
    },
    methods: {
        openModal() {
            $('#modalSendFile').modal({
                onApprove: function () {
                    return false;
                }
            }).modal('show');
        },
        isShowAttributes() {
            return this.type !== window.TYPESTATUS;
        },
        isValidForm() {
            if (this.type !== window.TYPESTATUS && !this.phone.trim()) {
                return false;
            }

            if (!this.selectedFileName) {
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
                $('#modalSendFile').modal('hide');
            } catch (err) {
                showErrorInfo(err)
            }
        },
        async submitApi() {
            this.loading = true;
            try {
                let payload = new FormData();
                payload.append("caption", this.caption)
                payload.append("phone", this.phone_id)
                payload.append("is_forwarded", this.is_forwarded)
                if (this.duration && this.duration > 0) {
                    payload.append("duration", this.duration)
                }
                payload.append("file", $("#file_file")[0].files[0])
                let response = await window.http.post(`/send/file`, payload)
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
            this.caption = '';
            this.phone = '';
            this.type = window.TYPEUSER;
            this.selectedFileName = null;
            this.is_forwarded = false;
            this.duration = 0;
            $("#file_file").val('');
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
            <div class="header">Enviar Arquivo</div>
            <div class="description">
                Envie qualquer arquivo de até
                <div class="ui blue horizontal label">{{ maxFileSize }}</div>
            </div>
        </div>
    </div>
    
    <!--  Modal SendFile  -->
    <div class="ui small modal" id="modalSendFile">
        <i class="close icon"></i>
        <div class="header">
            Enviar Arquivo
        </div>
        <div class="content">
            <form class="ui form">
                <FormRecipient v-model:type="type" v-model:phone="phone"/>
                
                <div class="field">
                    <label>Legenda</label>
                    <textarea v-model="caption" placeholder="Digite uma legenda (opcional)..."
                              aria-label="caption"></textarea>
                </div>
                <div class="field" v-if="isShowAttributes()">
                    <label>É Encaminhada</label>
                    <div class="ui toggle checkbox">
                        <input type="checkbox" aria-label="is forwarded" v-model="is_forwarded">
                        <label>Marcar arquivo como encaminhado</label>
                    </div>
                </div>
                <div class="field">
                    <label>Duração de Mensagem Temporária (segundos)</label>
                    <input v-model.number="duration" type="number" min="0" max="7776000" placeholder="0 (sem expiração), 24h a 90d" aria-label="duration"/>
                </div>
                <div class="field" style="padding-bottom: 30px">
                    <label>Arquivo</label>
                    <input type="file" style="display: none" id="file_file" @change="handleFileChange">
                    <label for="file_file" class="ui positive medium green left floated button" style="color: white">
                        <i class="ui upload icon"></i>
                        Carregar arquivo
                    </label>
                    <div v-if="selectedFileName" style="margin-top: 60px; clear: both;">
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