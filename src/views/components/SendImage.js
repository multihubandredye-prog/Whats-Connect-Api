import FormRecipient from "./generic/FormRecipient.js";

export default {
    name: 'SendImage',
    components: {
        FormRecipient
    },
    data() {
        return {
            phone: '',
            view_once: false,
            compress: false,
            caption: '',
            type: window.TYPEUSER,
            loading: false,
            selected_file: null,
            image_url: null,
            preview_url: null,
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

            if (!this.selected_file && !this.image_url) {
                return true;
            }
            
            return false;
        }
    },
    watch: {
        view_once(newValue) {
            // If view_once is set to true, set is_forwarded to false
            if (newValue === true) {
                this.is_forwarded = false;
                this.duration = 0;
            }
        }
    },
    methods: {
        openModal() {
            $('#modalSendImage').modal({
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

            if (!this.selected_file && !this.image_url) {
                return false;
            }

            // Validate duration if not view_once
            if (!this.view_once && this.duration !== 0 && (this.duration < 86400 || this.duration > 7776000)) {
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
                $('#modalSendImage').modal('hide');
            } catch (err) {
                showErrorInfo(err)
            }
        },
        async submitApi() {
            this.loading = true;
            try {
                let payload = new FormData();
                payload.append("phone", this.phone_id)
                payload.append("view_once", this.view_once)
                payload.append("compress", this.compress)
                payload.append("caption", this.caption)
                payload.append("is_forwarded", this.is_forwarded)
                if (this.duration && this.duration > 0) {
                    payload.append("duration", this.duration)
                }
                
                const fileInput = $("#file_image");
                if (fileInput.length > 0 && fileInput[0].files.length > 0) {
                    const file = fileInput[0].files[0];
                    payload.append('image', file);
                }
                if (this.image_url) {
                    payload.append('image_url', this.image_url)
                }
                
                let response = await window.http.post(`/send/image`, payload)
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
            this.view_once = false;
            this.compress = false;
            this.phone = '';
            this.caption = '';
            this.preview_url = null;
            this.selected_file = null;
            this.image_url = null;
            this.is_forwarded = false;
            this.duration = 0;
            $("#file_image").val('');
        },
        handleImageChange(event) {
            const file = event.target.files[0];
            if (file) {
                this.preview_url = URL.createObjectURL(file);
                // Add small delay to allow DOM update before scrolling
                setTimeout(() => {
                    const modalContent = document.querySelector('#modalSendImage .content');
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
            <div class="header">Enviar Imagem</div>
            <div class="description">
                Enviar imagem com
                <div class="ui blue horizontal label">jpg/jpeg/png</div>
                tipo
            </div>
        </div>
    </div>
    
    <!--  Modal SendImage  -->
    <div class="ui small modal" id="modalSendImage">
        <i class="close icon"></i>
        <div class="header">
            Enviar Imagem
        </div>
        <div class="content" style="max-height: 70vh; overflow-y: auto;">
            <form class="ui form">
                <FormRecipient v-model:type="type" v-model:phone="phone" :show-status="true"/>
                
                <div class="field">
                    <label>Legenda</label>
                    <textarea v-model="caption" type="text" placeholder="Olá, esta é a legenda da imagem"
                              aria-label="caption"></textarea>
                </div>
                <div class="field" v-if="isShowAttributes()">
                    <label>Ver Uma Vez</label>
                    <div class="ui toggle checkbox">
                        <input type="checkbox" aria-label="view once" v-model="view_once">
                        <label>Marque para ativar a visualização única</label>
                    </div>
                </div>
                <div class="field" v-if="isShowAttributes()">
                    <label>Comprimir</label>
                    <div class="ui toggle checkbox">
                        <input type="checkbox" aria-label="compress" v-model="compress">
                        <label>Marque para comprimir a imagem para um tamanho menor</label>
                    </div>
                </div>
                <div class="field" v-if="isShowAttributes() && !view_once">
                    <label>É Encaminhada</label>
                    <div class="ui toggle checkbox">
                        <input type="checkbox" aria-label="is forwarded" v-model="is_forwarded">
                        <label>Marcar imagem como encaminhada</label>
                    </div>
                </div>
                <div class="field">
                    <label>Duração de Mensagem Temporária (segundos)</label>
                    <input v-model.number="duration" type="number" min="0" max="7776000" placeholder="0 (sem expiração), 24h a 90d" aria-label="duration"/>
                </div>
                <div class="field">
                    <label>URL da Imagem</label>
                    <input type="text" v-model="image_url" placeholder="https://example.com/image.jpg"
                           aria-label="image_url"/>
                </div>
                <div style="text-align: left; font-weight: bold; margin: 10px 0;">ou você pode carregar imagem do seu dispositivo</div>
                <div class="field" style="padding-bottom: 30px">
                    <label>Imagem</label>
                    <input type="file" style="display: none" id="file_image" accept="image/png,image/jpg,image/jpeg" @change="handleImageChange"/>
                    <label for="file_image" class="ui positive medium green left floated button" style="color: white">
                        <i class="ui upload icon"></i>
                        Carregar imagem
                    </label>
                    <div v-if="preview_url" style="margin-top: 60px">
                        <img :src="preview_url" style="max-width: 100%; max-height: 300px; object-fit: contain" />
                    </div>
                </div>
            </form>
        </div>
        <div class="actions">
            <button class="ui approve positive right labeled icon button" 
                 :class="{'loading': this.loading, 'disabled': isSubmitButtonDisabled || loading}"
                 @click.prevent="handleSubmit">
                Enviar
                <i class="send icon"></i>
            </button>
        </div>
    </div>
    `
}