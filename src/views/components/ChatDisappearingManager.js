import FormRecipient from "./generic/FormRecipient.js";

export default {
    name: 'ChatDisappearingManager',
    components: {
        FormRecipient
    },
    data() {
        return {
            type: window.TYPEUSER,
            phone: '',
            timerSeconds: 86400, // Padrão para 24 horas
            loading: false,
        }
    },
    computed: {
        phone_id() {
            return this.phone + this.type;
        },
        timerLabel() {
            const labels = {
                0: 'Desativado',
                86400: '24 horas',
                604800: '7 dias',
                7776000: '90 dias'
            };
            return labels[this.timerSeconds] || 'Desconhecido';
        }
    },
    methods: {
        isValidForm() {
            const isPhoneValid = this.phone.trim().length > 0;
            return isPhoneValid;
        },
        openModal() {
            $('#modalChatDisappearing').modal({
                onApprove: function () {
                    return false;
                }
            }).modal('show');
        },
        async handleSubmit() {
            if (!this.isValidForm() || this.loading) {
                return;
            }
            try {
                const response = await this.submitApi();
                showSuccessInfo(response);
                $('#modalChatDisappearing').modal('hide');
            } catch (err) {
                showErrorInfo(err);
            }
        },
        async submitApi() {
            this.loading = true;
            try {
                const payload = {
                    timer_seconds: parseInt(this.timerSeconds)
                };

                const response = await window.http.post(`/chat/${this.phone_id}/disappearing`, payload);
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
            this.timerSeconds = 86400;
        },
    },
    template: `
    <div class="purple card" @click="openModal()" style="cursor: pointer">
        <div class="content">
            <a class="ui purple right ribbon label">Chat</a>
            <div class="header">Mensagens Temporárias</div>
            <div class="description">
                Definir temporizador de autoexclusão para mensagens de chat
            </div>
        </div>
    </div>
    
    <!--  Modal ChatDisappearing  -->
    <div class="ui small modal" id="modalChatDisappearing">
        <i class="close icon"></i>
        <div class="header">
            <i class="clock outline icon"></i> Mensagens Temporárias
        </div>
        <div class="content">
            <form class="ui form">
                <FormRecipient v-model:type="type" v-model:phone="phone" :show-status="false"/>
                <div class="field">
                    <label>Duração do Temporizador</label>
                    <select class="ui dropdown" v-model="timerSeconds">
                        <option :value="0">Desativado (desabilitado)</option>
                        <option :value="86400">24 horas</option>
                        <option :value="604800">7 dias</option>
                        <option :value="7776000">90 dias</option>
                    </select>
                </div>
                <div class="ui info message" v-if="timerSeconds > 0">
                    <i class="info circle icon"></i>
                    As mensagens desaparecerão após <strong>{{ timerLabel }}</strong>
                </div>
                <div class="ui warning message" v-else>
                    <i class="exclamation triangle icon"></i>
                    As mensagens temporárias serão <strong>desabilitadas</strong>
                </div>
            </form>
        </div>
        <div class="actions">
            <button class="ui approve positive right labeled icon button" 
                 :class="{'disabled': !isValidForm() || loading, 'loading': loading}"
                 @click.prevent="handleSubmit">
                {{ timerSeconds === 0 ? 'Desativar Temporizador' : 'Definir Temporizador' }}
                <i class="clock icon"></i>
            </button>
        </div>
    </div>
    `
}
