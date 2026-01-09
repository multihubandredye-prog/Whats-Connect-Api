// export Vue Component
import FormRecipient from "./generic/FormRecipient.js";

export default {
    name: 'SendPoll',
    components: {
        FormRecipient
    },
    data() {
        return {
            phone: '',
            type: window.TYPEUSER,
            loading: false,
            question: '',
            options: ['', ''],
            max_answer: 1,
            duration: 0,
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

            if (!this.question.trim()) {
                return true;
            }
            
            if (this.options.some(option => option.trim() === '')) {
                return true;
            }

            if (this.max_answer < 1 || this.max_answer > this.options.length) {
                return true;
            }
            
            return false;
        }
    },
    methods: {
        openModal() {
            $('#modalSendPoll').modal({
                onApprove: function () {
                    return false;
                }
            }).modal('show');
        },
        isValidForm() {
            if (this.type !== window.TYPESTATUS && !this.phone.trim()) {
                return false;
            }

            if (!this.question.trim()) {
                return false;
            }
            
            if (this.options.some(option => option.trim() === '')) {
                return false;
            }

            if (this.max_answer < 1 || this.max_answer > this.options.length) {
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
                window.showSuccessInfo(response)
                $('#modalSendPoll').modal('hide');
            } catch (err) {
                window.showErrorInfo(err)
            }
        },
        async submitApi() {
            this.loading = true;
            try {
                const payload = {
                    phone: this.phone_id,
                    question: this.question,
                    options: this.options,
                    max_answer: this.max_answer,
                    ...(this.duration && this.duration > 0 ? {duration: this.duration} : {})
                }
                const response = await window.http.post(`/send/poll`, payload)
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
            this.question = '';
            this.options = ['', ''];
            this.max_answer = 1;
            this.duration = 0;
        },
        addOption() {
            this.options.push('')
        },
        deleteOption(index) {
            this.options.splice(index, 1)
        }
    },
    template: `
    <div class="blue card" @click="openModal()" style="cursor: pointer">
        <div class="content">
            <a class="ui blue right ribbon label">Enviar</a>
            <div class="header">Enviar Enquete</div>
            <div class="description">
                Enviar uma enquete/votação com múltiplas opções
            </div>
        </div>
    </div>
    
    <!--  Modal SendPoll  -->
    <div class="ui small modal" id="modalSendPoll">
        <i class="close icon"></i>
        <div class="header">
            Enviar Enquete
        </div>
        <div class="content">
            <form class="ui form">
                <FormRecipient v-model:type="type" v-model:phone="phone"/>
                
                <div class="field">
                    <label>Pergunta</label>
                    <input v-model="question" type="text" placeholder="Por favor, insira a pergunta"
                           aria-label="poll question">
                </div>
                <div class="field">
                    <label>Opções</label>
                    <div style="display: flex; flex-direction: column; gap: 5px">
                        <div class="ui action input" :key="index" v-for="(option, index) in options">
                            <input type="text" placeholder="Opção..." v-model="options[index]"
                                   aria-label="poll option">
                            <button class="ui button" @click="deleteOption(index)" type="button">
                                <i class="minus circle icon"></i>
                            </button>
                        </div>
                        <div class="field">
                            <button class="mini ui primary button" @click="addOption" type="button">
                                <i class="plus icon"></i> Adicionar Opção
                            </button>
                        </div>
                    </div>
                </div>
                <div class="field">
                    <label>Máximo de Respostas Permitidas</label>
                    <input v-model.number="max_answer" type="number" placeholder="Máximo de respostas por usuário" 
                           aria-label="poll max answers" min="1" max="50">
                    <div class="ui pointing label">
                        Quantas opções cada usuário pode selecionar
                    </div>
                </div>
                <div class="field">
                    <label>Duração de Mensagem Temporária (segundos)</label>
                    <input v-model.number="duration" type="number" min="0" max="7776000" placeholder="0 (sem expiração), 24h a 90d" aria-label="duration"/>
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