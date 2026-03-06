import FormRecipient from "./generic/FormRecipient.js";

export default {
    name: 'SendMessage',
    components: {
        FormRecipient
    },
    data() {
        return {
            type: window.TYPEUSER,
            phone: '',
            text: '',
            reply_message_id: '',
            is_forwarded: false,
            mention_everyone: false,
            duration: 0,
            loading: false,
        }
    },
    computed: {
        phone_id() {
            return this.phone + this.type;
        },
        isSubmitButtonDisabled() {
            const isPhoneValid = this.type === window.TYPESTATUS || this.phone.trim().length > 0;
            const isMessageValid = this.text.trim().length > 0 && this.text.length <= 4096;

            return !isPhoneValid || !isMessageValid;
        }
    },
    methods: {
        openModal() {
            $('#modalSendMessage').modal({
                onApprove: function () {
                    return false;
                }
            }).modal('show');
        },
        isShowReplyId() {
            return this.type !== window.TYPESTATUS;
        },
        isGroup() {
            return this.type === window.TYPEGROUP;
        },
        isValidForm() {
            // Validate phone number is not empty except for status type
            const isPhoneValid = this.type === window.TYPESTATUS || this.phone.trim().length > 0;
            
            // Validate message is not empty and has reasonable length
            const isMessageValid = this.text.trim().length > 0 && this.text.length <= 4096;

            // Validate duration
            if (this.duration !== 0 && (this.duration < 86400 || this.duration > 7776000)) {
                showErrorInfo("Duração inválida. Use 0 para sem expiração, ou entre 24 horas (86400s) e 90 dias (7776000s).");
                return false;
            }

            return isPhoneValid && isMessageValid
        },
        async handleSubmit() {
            // Add validation check here to prevent submission when form is invalid
            if (!this.isValidForm() || this.loading) {
                return;
            }
            try {
                const response = await this.submitApi();
                showSuccessInfo(response);
                $('#modalSendMessage').modal('hide');
            } catch (err) {
                showErrorInfo(err);
            }
        },
        async submitApi() {
            this.loading = true;
            try {
                const payload = {
                    phone: this.phone_id,
                    message: this.text.trim(),
                    is_forwarded: this.is_forwarded
                };
                if (this.reply_message_id !== '') {
                    payload.reply_message_id = this.reply_message_id;
                }

                if (this.duration && this.duration > 0) {
                    payload.duration = this.duration;
                }

                // Add mentions if mention_everyone is checked (only for groups)
                if (this.mention_everyone && this.type === window.TYPEGROUP) {
                    payload.mentions = ["@everyone"];
                }

                const response = await window.http.post('/send/message', payload);
                this.handleReset();
                return response.data.message;
            } catch (error) {
                if (error.response?.data?.message) {
                    throw new Error(error.response.data.message);
                }
                throw error;
            } finally {
                this.loading = false;
            }
        },
        handleReset() {
            this.phone = '';
            this.text = '';
            this.reply_message_id = '';
            this.is_forwarded = false;
            this.mention_everyone = false;
            this.duration = 0;
        },
    },
    template: `
    <div class="blue card" @click="openModal()" style="cursor: pointer">
        <div class="content">
            <a class="ui blue right ribbon label">Enviar</a>
            <div class="header">Enviar Mensagem</div>
            <div class="description">
                Envie qualquer mensagem para um usuário ou grupo
            </div>
        </div>
    </div>
    
    <!--  Modal SendMessage  -->
    <div class="ui small modal" id="modalSendMessage">
        <i class="close icon"></i>
        <div class="header">
            Enviar Mensagem
        </div>
        <div class="content">
            <form class="ui form">
                <FormRecipient v-model:type="type" v-model:phone="phone" :show-status="true"/>
                <div class="field" v-if="isShowReplyId()">
                    <label>ID da Mensagem de Resposta</label>
                    <input v-model="reply_message_id" type="text"
                           placeholder="Opcional: 57D29F74B7FC62F57D8AC2C840279B5B/3EB0288F008D32FCD0A424"
                           aria-label="reply_message_id">
                </div>
                <div class="field">
                    <label>Mensagem</label>
                    <textarea v-model="text" placeholder="Olá, este é o texto da mensagem"
                              aria-label="message"></textarea>
                </div>
                <div class="field" v-if="isShowReplyId()">
                    <label>É Encaminhada</label>
                    <div class="ui toggle checkbox">
                        <input type="checkbox" aria-label="is forwarded" v-model="is_forwarded">
                        <label>Marcar mensagem como encaminhada</label>
                    </div>
                </div>
                <div class="field" v-if="isGroup()">
                    <label>Mencionar Todos</label>
                    <div class="ui toggle checkbox">
                        <input type="checkbox" aria-label="mention everyone" v-model="mention_everyone">
                        <label>Mencionar todos os participantes do grupo (@todos)</label>
                    </div>
                </div>
                <div class="field">
                    <label>Duração de Desaparecimento (segundos)</label>
                    <input v-model.number="duration" type="number" min="0" max="7776000" placeholder="0 (sem expiração), 24h a 90d" aria-label="duration"/>
                </div>
            </form>
        </div>
        <div class="actions">
            <button class="ui approve positive right labeled icon button" 
                 :class="{'disabled': isSubmitButtonDisabled || loading}"
                 @click.prevent="handleSubmit">
                Enviar
                <i class="send icon"></i>
            </button>
        </div>
    </div>
    `
}