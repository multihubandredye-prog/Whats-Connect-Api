export default {
    name: 'SendPresence',
    data() {
        return {
            type: 'available',
            loading: false,
        }
    },
    methods: {
        openModal() {
            $('#modalSendPresence').modal({
                onApprove: function () {
                    return false;
                }
            }).modal('show');
        },
        async handleSubmit() {
            if (this.loading) {
                return;
            }

            try {
                let response = await this.submitApi()
                showSuccessInfo(response)
                $('#modalSendPresence').modal('hide');
            } catch (err) {
                showErrorInfo(err)
            }
        },
        async submitApi() {
            this.loading = true;
            try {
                let payload = {
                    type: this.type
                }
                let response = await window.http.post(`/send/presence`, payload)
                return response.data.message;
            } catch (error) {
                if (error.response) {
                    throw new Error(error.response.data.message);
                }
                throw new Error(error.message);
            } finally {
                this.loading = false;
            }
        }
    },
    template: `
    <div class="blue card" @click="openModal()" style="cursor: pointer">
        <div class="content">
            <a class="ui blue right ribbon label">Enviar</a>
            <div class="header">Enviar Presença</div>
            <div class="description">
                Definir <div class="ui green horizontal label">disponível</div> ou <div class="ui grey horizontal label">indisponível</div>
            </div>
        </div>
    </div>
    
    <!--  Modal SendPresence  -->
    <div class="ui small modal" id="modalSendPresence">
        <i class="close icon"></i>
        <div class="header">
            Enviar Presença
        </div>
        <div class="content">
            <form class="ui form">
                <div class="field">
                    <label>Status de Presença</label>
                    <select v-model="type" class="ui dropdown">
                        <option value="available">Disponível</option>
                        <option value="unavailable">Indisponível</option>
                    </select>
                </div>
            </form>
        </div>
        <div class="actions">
            <button class="ui approve positive right labeled icon button" 
                 :class="{'loading': loading, 'disabled': loading}"
                 @click.prevent="handleSubmit">
                Enviar
                <i class="send icon"></i>
            </button>
        </div>
    </div>
    `
}