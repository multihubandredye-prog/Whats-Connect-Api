import FormRecipient from "./generic/FormRecipient.js";

export default {
    name: 'SendSticker',
    components: {
        FormRecipient
    },
    data() {
        return {
            phone: '',
            type: window.TYPEUSER,
            loading: false,
            selected_file: null,
            sticker_url: null,
            preview_url: null,
            is_forwarded: false,
            duration: 0
        }
    },
    computed: {
        phone_id() {
            return this.phone + this.type;
        },
    },
    methods: {
        openModal() {
            $('#modalSendSticker').modal({
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

            if (!this.selected_file && !this.sticker_url) {
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
                $('#modalSendSticker').modal('hide');
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
                if (this.duration && this.duration > 0) {
                    payload.append("duration", this.duration)
                }
                
                const fileInput = $("#file_sticker");
                if (fileInput.length > 0 && fileInput[0].files.length > 0) {
                    const file = fileInput[0].files[0];
                    payload.append('sticker', file);
                }
                if (this.sticker_url) {
                    payload.append('sticker_url', this.sticker_url)
                }
                
                let response = await window.http.post(`/send/sticker`, payload)
                this.handleReset();
                return response.data.message;
            } catch (error) {
                if (error.response) {
                    throw new Error(error.response.data.message);
                }
                throw new Error(error.message);
            }
            finally {
                this.loading = false;
            }
        },
        handleReset() {
            this.phone = '';
            this.preview_url = null;
            this.selected_file = null;
            this.sticker_url = null;
            this.is_forwarded = false;
            this.duration = 0;
            $("#file_sticker").val('');
        },
        handleStickerChange(event) {
            const file = event.target.files[0];
            if (file) {
                this.preview_url = URL.createObjectURL(file);
                // Add small delay to allow DOM update before scrolling
                setTimeout(() => {
                    const modalContent = document.querySelector('#modalSendSticker .content');
                    if (modalContent) {
                        modalContent.scrollTop = modalContent.scrollHeight;
                    }
                    this.selected_file = file.name;
                }, 100);
            }
        }
    },
    template: `
    <div class="blue card" @click="openModal()" style="cursor:pointer;">
        <div class="content">
            <a class="ui blue right ribbon label">Enviar</a>
            <div class="header">Enviar Sticker</div>
            <div class="description">
                Enviar sticker com conversão automática para formato WebP
                <div class="ui blue horizontal label">jpg/jpeg/png/webp/gif</div>
            </div>
        </div>
    </div>
    
    <!--  Modal SendSticker  -->
    <div class="ui small modal" id="modalSendSticker">
        <i class="close icon"></i>
        <div class="header">
            Enviar Sticker
        </div>
        <div class="content" style="max-height: 70vh; overflow-y: auto;">
            <form class="ui form">
                <FormRecipient v-model:type="type" v-model:phone="phone" :show-status="true"/>
                
                <div class="ui info message">
                    <div class="header">Informações do Sticker</div>
                    <ul class="list">
                        <li>Imagens serão automaticamente convertidas para o formato de sticker WebP</li>
                        <li>Tamanho máximo do sticker é 512x512 pixels (redimensionamento automático)</li>
                        <li>Suporta formatos JPG, JPEG, PNG, WebP e GIF</li>
                        <li>Fundos transparentes são preservados para imagens PNG</li>
                    </ul>
                </div>
                
                <div class="field" v-if="isShowAttributes()">
                    <label>É Encaminhada</label>
                    <div class="ui toggle checkbox">
                        <input type="checkbox" aria-label="is forwarded" v-model="is_forwarded">
                        <label>Marcar sticker como encaminhado</label>
                    </div>
                </div>
                <div class="field">
                    <label>Duração de Mensagem Temporária (segundos)</label>
                    <input v-model.number="duration" type="number" min="0" max="7776000" placeholder="0 (sem expiração), 24h a 90d" aria-label="duration"/>
                </div>
                <div class="field">
                    <label>URL do Sticker</label>
                    <input type="text" v-model="sticker_url" placeholder="https://example.com/sticker.png"
                           aria-label="sticker_url"/>
                </div>
                <div style="text-align: left; font-weight: bold; margin: 10px 0;">ou você pode carregar sticker do seu dispositivo</div>
                <div class="field" style="padding-bottom: 30px">
                    <label>Imagem do Sticker</label>
                    <input type="file" style="display: none" id="file_sticker" accept="image/png,image/jpg,image/jpeg,image/webp,image/gif" @change="handleStickerChange"/>
                    <label for="file_sticker" class="ui positive medium blue left floated button" style="color: white">
                        <i class="ui upload icon"></i>
                        Carregar sticker
                    </label>
                    <div v-if="preview_url" style="margin-top: 60px">
                        <div class="ui segment">
                            <img :src="preview_url" style="max-width: 100%; max-height: 300px; object-fit: contain" />
                            <div class="ui top attached label">Pré-visualização (será convertido para sticker WebP)</div>
                        </div>
                    </div>
                </div>
            </form>
        </div>
        <div class="actions">
            <button class="ui approve positive right labeled icon button" 
                 :class="{'loading': this.loading, 'disabled': !isValidForm() || loading}"
                 @click.prevent="handleSubmit">
                Enviar
                <i class="send icon"></i>
            </button>
        </div>
    </div>
    `
}