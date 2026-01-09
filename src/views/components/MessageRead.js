import FormRecipient from "./generic/FormRecipient.js";

export default {
    name: 'ReadMessage',
    components: {
        FormRecipient
    },
    data() {
        return {
            type: window.TYPEUSER,
            phone: '',
            message_id: '',
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
            $('#modalMessageRead').modal({
                onApprove: function () {
                    return false;
                }
            }).modal('show');
        },
        isValidForm() {
            if (this.type !== window.TYPESTATUS && !this.phone.trim()) {
                return false;
            }

            if (!this.message_id.trim()) {
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
                $('#modalMessageRead').modal('hide');
            } catch (err) {
                showErrorInfo(err)
            }
        },
        async submitApi() {
            this.loading = true;
            try {
                const payload = {phone: this.phone_id}

                let response = await window.http.post(`/message/${this.message_id}/read`, payload)
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
            this.type = window.TYPEUSER;
            this.phone = '';
            this.message_id = '';
            this.loading = false;
        },
    },
    template: `
    <div class="red card" @click="openModal()" style="cursor: pointer">
        <div class="content">
            <a class="ui red right ribbon label">Mensagem</a>
            <div class="header">Marcar como Lida</div>
            <div class="description">
                Marcar uma mensagem como lida em um chat
            </div>
        </div>
    </div>
        
        <!--  Modal MessageRead  -->
    <div class="ui small modal" id="modalMessageRead">
        <i class="close icon"></i>
        <div class="header">
            Marcar Mensagem como Lida
        </div>
        <div class="content">
            <form class="ui form">
                <FormRecipient v-model:type="type" v-model:phone="phone"/>
                
                <div class="field">
                    <label>ID da Mensagem</label>
                    <input v-model="message_id" type="text" placeholder="Por favor, insira o ID da mensagem para marcar como lida"
                           aria-label="message id">
                </div>
            </form>
        </div>
        <div class="actions">
            <button class="ui approve positive right labeled icon button" :class="{'loading': this.loading, 'disabled': !isValidForm() || loading}"
                 @click.prevent="handleSubmit">
                Marcar como Lida
                <i class="check icon"></i>
            </button>
        </div>
    </div>
    `
}