import FormRecipient from "./generic/FormRecipient.js";

export default {
    name: 'ReactMessage',
    components: {
        FormRecipient
    },
    data() {
        return {
            type: window.TYPEUSER,
            phone: '',
            message_id: '',
            emoji: '',
            loading: false,
        }
    },
    computed: {
        phone_id() {
            return this.phone + this.type;
        }
    },
    methods: {
        openModal() {
            $('#modalMessageReaction').modal({
                onApprove: function () {
                    return false;
                }
            }).modal('show');
        },
        isValidForm() {
            if (this.type !== window.TYPESTATUS && !this.phone.trim()) {
                return false;
            }

            if (!this.message_id.trim() || !this.emoji.trim()) {
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
                $('#modalMessageReaction').modal('hide');
            } catch (err) {
                showErrorInfo(err)
            }
        },
        async submitApi() {
            this.loading = true;
            try {
                const payload = {phone: this.phone_id, emoji: this.emoji}
                let response = await window.http.post(`/message/${this.message_id}/reaction`, payload)
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
            this.message_id = '';
            this.emoji = '';
            this.type = window.TYPEUSER;
        },
    },
    template: `
    <div class="red card" @click="openModal()" style="cursor: pointer">
        <div class="content">
            <a class="ui red right ribbon label">Mensagem</a>
            <div class="header">Reagir à Mensagem</div>
            <div class="description">
                 reaja a qualquer mensagem em chat privado ou em grupo
            </div>
        </div>
    </div>
    
    
    <!--  Modal MessageReaction  -->
    <div class="ui small modal" id="modalMessageReaction">
        <i class="close icon"></i>
        <div class="header">
             Reagir à Mensagem
        </div>
        <div class="content">
            <form class="ui form">
                <FormRecipient v-model:type="type" v-model:phone="phone"/>
                
                <div class="field">
                    <label>ID da Mensagem</label>
                    <input v-model="message_id" type="text" placeholder="Por favor, insira o ID da sua mensagem"
                           aria-label="message id">
                </div>
                <div class="field">
                    <label>Emoji</label>
                    <input v-model="emoji" type="text" placeholder="Por favor, insira um emoji"
                           aria-label="message id">
                </div>
            </form>
        </div>
        <div class="actions">
            <button class="ui approve positive right labeled icon button" :class="{'loading': this.loading, 'disabled': !this.isValidForm() || this.loading}"
                 @click="handleSubmit">
                Enviar
                <i class="send icon"></i>
            </button>
        </div>
    </div>
    `
}